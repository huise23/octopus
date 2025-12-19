package op

import (
	"context"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/bestruirui/octopus/internal/utils/log"
)

var channelCache = cache.New[int, model.Channel](16)

func ChannelList(ctx context.Context) ([]model.Channel, error) {
	channels := make([]model.Channel, 0, channelCache.Len())
	for _, channel := range channelCache.GetAll() {
		channels = append(channels, channel)
	}
	return channels, nil
}

func ChannelCreate(channel *model.Channel, ctx context.Context) error {
	if err := db.GetDB().WithContext(ctx).Create(channel).Error; err != nil {
		return err
	}
	channelCache.Set(channel.ID, *channel)
	return nil
}

func ChannelUpdate(channel *model.Channel, ctx context.Context) error {
	oldChannel, ok := channelCache.Get(channel.ID)
	if !ok {
		return fmt.Errorf("channel not found")
	}
	if oldChannel == *channel {
		return nil
	}

	// 获取旧渠道的模型列表
	oldModels := strings.Split(oldChannel.Model+","+oldChannel.CustomModel, ",")
	oldModelsSet := make(map[string]bool)
	for _, modelName := range oldModels {
		if modelName != "" {
			oldModelsSet[modelName] = true
		}
	}

	// 获取新渠道的模型列表
	newModels := strings.Split(channel.Model+","+channel.CustomModel, ",")
	newModelsSet := make(map[string]bool)
	for _, modelName := range newModels {
		if modelName != "" {
			newModelsSet[modelName] = true
		}
	}

	// 找出被移除的模型
	removedModels := make([]string, 0)
	for modelName := range oldModelsSet {
		if !newModelsSet[modelName] {
			removedModels = append(removedModels, modelName)
		}
	}

	if err := db.GetDB().WithContext(ctx).Save(channel).Error; err != nil {
		return err
	}
	channelCache.Set(channel.ID, *channel)

	// 检查并删除不再使用的模型
	for _, modelName := range removedModels {
		isUsedByOtherChannel := false
		for _, otherChannel := range channelCache.GetAll() {
			if otherChannel.ID == channel.ID {
				continue
			}
			otherChannelModels := strings.Split(otherChannel.Model+","+otherChannel.CustomModel, ",")
			for _, otherModelName := range otherChannelModels {
				if otherModelName == modelName {
					isUsedByOtherChannel = true
					break
				}
			}
			if isUsedByOtherChannel {
				break
			}
		}

		// 如果该模型没有被其他渠道使用，则删除
		if !isUsedByOtherChannel {
			if err := LLMDelete(modelName, ctx); err != nil {
				log.Errorf("failed to delete unused model %s: %v", modelName, err)
			}
		}
	}

	return nil
}

func ChannelDel(id int, ctx context.Context) error {
	channel, ok := channelCache.Get(id)
	if !ok {
		return fmt.Errorf("channel not found")
	}

	// 获取该渠道的所有模型
	deletedChannelModels := strings.Split(channel.Model+","+channel.CustomModel, ",")
	deletedChannelModelsSet := make(map[string]bool)
	for _, modelName := range deletedChannelModels {
		if modelName != "" {
			deletedChannelModelsSet[modelName] = true
		}
	}

	// 开启事务
	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取所有受影响的 GroupID，用于刷新缓存
	var affectedGroupIDs []int
	if err := tx.Model(&model.GroupItem{}).
		Where("channel_id = ?", id).
		Pluck("group_id", &affectedGroupIDs).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get affected groups: %w", err)
	}

	// 删除所有引用该渠道的 GroupItem
	if err := tx.Where("channel_id = ?", id).Delete(&model.GroupItem{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete group items: %w", err)
	}

	// 删除统计数据
	if err := tx.Where("channel_id = ?", id).Delete(&model.StatsChannel{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete channel stats: %w", err)
	}

	// 删除渠道
	if err := tx.Delete(&model.Channel{}, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 删除缓存
	channelCache.Del(id)
	StatsChannelDel(id)

	// 刷新受影响的分组缓存
	for _, groupID := range affectedGroupIDs {
		if err := groupRefreshCacheByID(groupID, ctx); err != nil {
			log.Errorf("failed to refresh group cache for group %d: %v", groupID, err)
		}
	}

	// 检查并删除仅该渠道使用的模型
	for modelName := range deletedChannelModelsSet {
		isUsedByOtherChannel := false
		for _, otherChannel := range channelCache.GetAll() {
			if otherChannel.ID == id {
				continue
			}
			otherChannelModels := strings.Split(otherChannel.Model+","+otherChannel.CustomModel, ",")
			for _, otherModelName := range otherChannelModels {
				if otherModelName == modelName {
					isUsedByOtherChannel = true
					break
				}
			}
			if isUsedByOtherChannel {
				break
			}
		}

		// 如果该模型没有被其他渠道使用，则删除
		if !isUsedByOtherChannel {
			if err := LLMDelete(modelName, ctx); err != nil {
				log.Errorf("failed to delete unused model %s: %v", modelName, err)
			}
		}
	}

	return nil
}

func ChannelLLMList(ctx context.Context) ([]model.LLMChannel, error) {
	models := []model.LLMChannel{}
	for _, channel := range channelCache.GetAll() {
		if channel.Enabled {
			modelNames := strings.Split(channel.Model+","+channel.CustomModel, ",")
			for _, modelName := range modelNames {
				if modelName == "" {
					continue
				}
				models = append(models, model.LLMChannel{
					Name:        modelName,
					ChannelID:   channel.ID,
					ChannelName: channel.Name,
				})
			}
		}
	}
	return models, nil
}

func ChannelGet(id int, ctx context.Context) (*model.Channel, error) {
	channel, ok := channelCache.Get(id)
	if !ok {
		return nil, fmt.Errorf("channel not found")
	}
	return &channel, nil
}

func channelRefreshCache(ctx context.Context) error {
	channels := []model.Channel{}
	if err := db.GetDB().WithContext(ctx).Find(&channels).Error; err != nil {
		log.Errorf("failed to get channels: %v", err)
		return err
	}
	for _, channel := range channels {
		channelCache.Set(channel.ID, channel)
	}
	return nil
}
