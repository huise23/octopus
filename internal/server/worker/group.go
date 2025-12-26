package worker

import (
	"context"
	"strings"

	"github.com/bestruirui/octopus/internal/matcher"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

func AutoAddGroupItem(id int, ctx context.Context) error {
	group, err := op.GroupGet(id, ctx)
	if err != nil {
		return err
	}
	channels, err := op.ChannelList(ctx)
	if err != nil {
		return err
	}

	// 初始化关键字匹配器
	keywordMatcher := matcher.NewKeywordMatcher()

	keys := make([]model.GroupIDAndLLMName, 0)
	for _, channel := range channels {
		modelNames := strings.Split(channel.Model+","+channel.CustomModel, ",")
		for _, modelName := range modelNames {
			if modelName == "" {
				continue
			}

			var matched bool

			// 使用新的匹配引擎（优先使用关键字匹配）
			if group.MatchMode != model.GroupMatchModeNameOnly {
				matched = keywordMatcher.MatchModel(modelName, *group)
				if matched {
					log.Infof("AutoAddGroupItem: model [%s] matched group [%s] via keyword matching (mode: %d)", modelName, group.Name, group.MatchMode)
				}
			} else {
				// 保持原有逻辑兼容性（仅分组名称匹配）
				matched = strings.Contains(strings.ToLower(modelName), strings.ToLower(group.Name))
				if matched {
					log.Infof("AutoAddGroupItem: model [%s] matched group [%s] via name matching", modelName, group.Name)
				}
			}

			if matched {
				keys = append(keys, model.GroupIDAndLLMName{ChannelID: channel.ID, ModelName: modelName})
			}
		}
	}

	if err := op.GroupItemBatchAdd(id, keys, ctx); err != nil {
		return err
	}
	return nil
}
