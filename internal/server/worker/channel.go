package worker

import (
	"context"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/matcher"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/utils/log"
)

func AutoGroup(channelID int, channelName, channelModel, customModel string, autoGroupType model.AutoGroupType) {
	if autoGroupType == model.AutoGroupTypeNone {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	groups, err := op.GroupList(ctx)
	if err != nil {
		log.Errorf("get group list failed: %v", err)
		return
	}

	// 初始化关键字匹配器
	keywordMatcher := matcher.NewKeywordMatcher()

	modelNames := strings.Split(channelModel+","+customModel, ",")
	for _, modelName := range modelNames {
		if modelName == "" {
			continue
		}

		for _, group := range groups {
			var matched bool

			// 使用新的匹配引擎（优先使用关键字匹配）
			if group.MatchMode != model.GroupMatchModeNameOnly {
				matched = keywordMatcher.MatchModel(modelName, group)
				if matched {
					log.Infof("model [%s] matched group [%s] via keyword matching (mode: %d)", modelName, group.Name, group.MatchMode)
				}
			} else {
				// 保持原有逻辑兼容性（仅分组名称匹配）
				switch autoGroupType {
				case model.AutoGroupTypeExact:
					matched = strings.EqualFold(modelName, group.Name)
				case model.AutoGroupTypeFuzzy:
					matched = strings.Contains(strings.ToLower(modelName), strings.ToLower(group.Name))
				}
				if matched {
					log.Infof("model [%s] matched group [%s] via name matching (type: %d)", modelName, group.Name, autoGroupType)
				}
			}

			if matched {
				// 检查是否已存在
				exists := false
				for _, item := range group.Items {
					if item.ChannelID == channelID && item.ModelName == modelName {
						exists = true
						break
					}
				}
				if exists {
					log.Debugf("model [%s] already exists in group [%s], skipping", modelName, group.Name)
					break
				}

				// 添加到分组
				err := op.GroupItemAdd(&model.GroupItem{
					GroupID:   group.ID,
					ChannelID: channelID,
					ModelName: modelName,
					Priority:  len(group.Items) + 1,
					Weight:    1,
				}, ctx)
				if err != nil {
					log.Errorf("add channel %s model %s to group %s failed: %v", channelName, modelName, group.Name, err)
				} else {
					log.Infof("channel %s: model [%s] added to group [%s]", channelName, modelName, group.Name)
				}
				break
			}
		}
	}
}

func CheckAndAddLLMPrice(channelModel, customModel string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		modelNames := strings.Split(channelModel+","+customModel, ",")
		var newModels []string
		for _, modelName := range modelNames {
			if modelName == "" {
				continue
			}
			modelPrice := price.GetLLMPrice(modelName)
			if modelPrice == nil {
				newModels = append(newModels, modelName)
			}
		}
		if len(newModels) > 0 {
			log.Infof("create models: %v", newModels)
			if err := op.LLMBatchCreate(newModels, ctx); err != nil {
				log.Errorf("failed to batch create models: %v", err)
			}
		}
	}()
}
