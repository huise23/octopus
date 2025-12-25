# 分组管理关键字匹配功能规划文档

## 项目概述

基于 Octopus LLM 网关系统，增强分组管理功能，新增关键字匹配机制（支持正则表达式），并修复渠道模型移除后分组仍调用的问题。

## 需求分析

### 1. 核心需求
- **关键字匹配功能**：除了现有的分组名称匹配，增加关键字匹配机制
- **正则表达式支持**：支持正则表达式进行复杂模式匹配
- **自动分组增强**：渠道选择模型后可以自动分组到匹配的分组中
- **模型移除修复**：修复渠道移除模型后，分组中已不显示但实际还会调用的问题

### 2. 当前系统分析

#### 现有分组匹配机制
- **精确匹配**（AutoGroupTypeExact）：`strings.EqualFold(modelName, group.Name)`
- **模糊匹配**（AutoGroupTypeFuzzy）：`strings.Contains(strings.ToLower(modelName), strings.ToLower(group.Name))`
- **匹配位置**：`internal/server/worker/channel.go:AutoGroup()` 函数

#### 现有问题识别
1. **模型移除问题**：
   - 位置：`internal/op/channel.go:ChannelUpdate()` 函数
   - 问题：只删除了 LLM 价格信息，但未删除分组中的 GroupItem
   - 影响：分组中仍显示已移除的模型，可能导致调用失败

2. **匹配机制限制**：
   - 只支持分组名称匹配
   - 无法支持复杂的匹配规则
   - 缺乏正则表达式支持

## 技术方案设计

### 1. 数据库结构扩展

#### 1.1 Group 表结构扩展
```sql
ALTER TABLE groups ADD COLUMN keywords TEXT DEFAULT '';
ALTER TABLE groups ADD COLUMN match_mode INT DEFAULT 0;
```

#### 1.2 新增字段说明
- `keywords`：关键字列表，JSON 格式存储
- `match_mode`：匹配模式
  - 0: 仅分组名称匹配（保持兼容）
  - 1: 仅关键字匹配
  - 2: 分组名称 + 关键字匹配

### 2. 模型结构更新

#### 2.1 Group 结构体扩展
```go
// internal/model/group.go
type Group struct {
    ID        int           `json:"id" gorm:"primaryKey"`
    Name      string        `json:"name" gorm:"unique;not null"`
    Mode      GroupMode     `json:"mode" gorm:"not null"`
    Keywords  string        `json:"keywords" gorm:"type:text;default:''"`     // JSON 格式的关键字列表
    MatchMode GroupMatchMode `json:"match_mode" gorm:"default:0"`             // 匹配模式
    Items     []GroupItem   `json:"items,omitempty" gorm:"foreignKey:GroupID"`
}

// 新增匹配模式枚举
type GroupMatchMode int

const (
    GroupMatchModeNameOnly    GroupMatchMode = 0 // 仅分组名称匹配
    GroupMatchModeKeywordOnly GroupMatchMode = 1 // 仅关键字匹配
    GroupMatchModeBoth        GroupMatchMode = 2 // 分组名称 + 关键字匹配
)

// 关键字配置结构
type GroupKeyword struct {
    Pattern string `json:"pattern"` // 匹配模式（支持正则）
    Type    string `json:"type"`    // 匹配类型：exact, fuzzy, regex
}
```

#### 2.2 请求结构体更新
```go
// internal/model/group.go
type GroupUpdateRequest struct {
    ID            int                      `json:"id" binding:"required"`
    Name          *string                  `json:"name,omitempty"`
    Mode          *GroupMode               `json:"mode,omitempty"`
    Keywords      *string                  `json:"keywords,omitempty"`      // 新增
    MatchMode     *GroupMatchMode          `json:"match_mode,omitempty"`    // 新增
    ItemsToAdd    []GroupItemAddRequest    `json:"items_to_add,omitempty"`
    ItemsToUpdate []GroupItemUpdateRequest `json:"items_to_update,omitempty"`
    ItemsToDelete []int                    `json:"items_to_delete,omitempty"`
}
```

### 3. 核心功能实现

#### 3.1 关键字匹配引擎
```go
// internal/matcher/keyword_matcher.go
package matcher

import (
    "encoding/json"
    "regexp"
    "strings"
    "github.com/bestruirui/octopus/internal/model"
)

type KeywordMatcher struct {
    compiledRegexes map[string]*regexp.Regexp
}

func NewKeywordMatcher() *KeywordMatcher {
    return &KeywordMatcher{
        compiledRegexes: make(map[string]*regexp.Regexp),
    }
}

func (km *KeywordMatcher) MatchModel(modelName string, group model.Group) bool {
    // 根据匹配模式进行匹配
    switch group.MatchMode {
    case model.GroupMatchModeNameOnly:
        return km.matchByName(modelName, group.Name)
    case model.GroupMatchModeKeywordOnly:
        return km.matchByKeywords(modelName, group.Keywords)
    case model.GroupMatchModeBoth:
        return km.matchByName(modelName, group.Name) || km.matchByKeywords(modelName, group.Keywords)
    default:
        return km.matchByName(modelName, group.Name) // 默认兼容模式
    }
}

func (km *KeywordMatcher) matchByName(modelName, groupName string) bool {
    // 保持现有的模糊匹配逻辑
    return strings.Contains(strings.ToLower(modelName), strings.ToLower(groupName))
}

func (km *KeywordMatcher) matchByKeywords(modelName, keywordsJSON string) bool {
    if keywordsJSON == "" {
        return false
    }

    var keywords []model.GroupKeyword
    if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
        return false
    }

    for _, keyword := range keywords {
        if km.matchSingleKeyword(modelName, keyword) {
            return true
        }
    }
    return false
}

func (km *KeywordMatcher) matchSingleKeyword(modelName string, keyword model.GroupKeyword) bool {
    switch keyword.Type {
    case "exact":
        return strings.EqualFold(modelName, keyword.Pattern)
    case "fuzzy":
        return strings.Contains(strings.ToLower(modelName), strings.ToLower(keyword.Pattern))
    case "regex":
        return km.matchRegex(modelName, keyword.Pattern)
    default:
        return km.matchByName(modelName, keyword.Pattern) // 默认模糊匹配
    }
}

func (km *KeywordMatcher) matchRegex(modelName, pattern string) bool {
    regex, exists := km.compiledRegexes[pattern]
    if !exists {
        var err error
        regex, err = regexp.Compile(pattern)
        if err != nil {
            return false // 正则表达式编译失败
        }
        km.compiledRegexes[pattern] = regex
    }
    return regex.MatchString(modelName)
}
```

#### 3.2 自动分组功能增强
```go
// internal/server/worker/channel.go
// 更新 AutoGroup 函数

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
    matcher := matcher.NewKeywordMatcher()

    modelNames := strings.Split(channelModel+","+customModel, ",")
    for _, modelName := range modelNames {
        if modelName == "" {
            continue
        }

        for _, group := range groups {
            var matched bool

            // 使用新的匹配引擎
            if group.MatchMode != model.GroupMatchModeNameOnly {
                matched = matcher.MatchModel(modelName, group)
            } else {
                // 保持原有逻辑兼容性
                switch autoGroupType {
                case model.AutoGroupTypeExact:
                    matched = strings.EqualFold(modelName, group.Name)
                case model.AutoGroupTypeFuzzy:
                    matched = strings.Contains(strings.ToLower(modelName), strings.ToLower(group.Name))
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
                    log.Infof("channel %s: model [%s] added to group [%s] via keyword matching", channelName, modelName, group.Name)
                }
                break
            }
        }
    }
}
```

#### 3.3 渠道模型移除问题修复
```go
// internal/op/channel.go
// 在 ChannelUpdate 函数中添加 GroupItem 清理逻辑

func ChannelUpdate(channel *model.Channel, ctx context.Context) error {
    oldChannel, ok := channelCache.Get(channel.ID)
    if !ok {
        return fmt.Errorf("channel not found")
    }
    if oldChannel == *channel {
        return nil
    }

    // 检查是否是启用状态的变更
    enabledChanged := oldChannel.Enabled != channel.Enabled

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

    // 【新增】删除分组中对应的 GroupItem
    if len(removedModels) > 0 {
        removedKeys := make([]model.GroupIDAndLLMName, len(removedModels))
        for i, modelName := range removedModels {
            removedKeys[i] = model.GroupIDAndLLMName{
                ChannelID: channel.ID,
                ModelName: modelName,
            }
        }

        if err := GroupItemBatchDelByChannelAndModels(removedKeys, ctx); err != nil {
            log.Errorf("failed to remove group items for removed models: %v", err)
        } else {
            log.Infof("removed %d group items for channel %s removed models", len(removedKeys), channel.Name)
        }
    }

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

    // 如果渠道启用状态发生变化，需要刷新所有相关分组的缓存
    if enabledChanged {
        if err := refreshGroupsByChannelID(channel.ID, ctx); err != nil {
            log.Errorf("failed to refresh groups for channel %d: %v", channel.ID, err)
        }
    }

    return nil
}
```

### 4. API 接口扩展

#### 4.1 分组管理接口更新
```go
// internal/server/handlers/group.go

// 创建分组接口扩展
type GroupCreateRequest struct {
    Name      string             `json:"name" binding:"required"`
    Mode      model.GroupMode    `json:"mode" binding:"required"`
    Keywords  string             `json:"keywords,omitempty"`      // 新增
    MatchMode model.GroupMatchMode `json:"match_mode,omitempty"`  // 新增
}

// 分组列表响应扩展
type GroupListResponse struct {
    ID        int                    `json:"id"`
    Name      string                 `json:"name"`
    Mode      model.GroupMode        `json:"mode"`
    Keywords  []model.GroupKeyword   `json:"keywords"`      // 解析后的关键字列表
    MatchMode model.GroupMatchMode   `json:"match_mode"`
    Items     []model.GroupItem      `json:"items"`
}

// 关键字测试接口
func GroupTestKeywords(c *gin.Context) {
    var req struct {
        ModelName string                 `json:"model_name" binding:"required"`
        Keywords  []model.GroupKeyword   `json:"keywords" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    matcher := matcher.NewKeywordMatcher()
    keywordsJSON, _ := json.Marshal(req.Keywords)

    matched := matcher.matchByKeywords(req.ModelName, string(keywordsJSON))

    resp.Success(c, gin.H{
        "matched": matched,
        "model_name": req.ModelName,
        "keywords": req.Keywords,
    })
}
```

#### 4.2 新增 API 端点
```
POST   /api/v1/group/test-keywords    - 测试关键字匹配
GET    /api/v1/group/match-preview    - 预览匹配结果
```

### 5. 前端界面设计

#### 5.1 分组管理界面扩展
- **关键字配置区域**：
  - 关键字列表管理
  - 匹配类型选择（精确/模糊/正则）
  - 实时预览匹配结果

- **匹配模式选择**：
  - 单选框：仅分组名称/仅关键字/两者结合

- **测试功能**：
  - 输入模型名称测试匹配结果
  - 显示匹配的关键字和规则

#### 5.2 界面交互流程
1. 用户创建/编辑分组时可配置关键字
2. 支持添加多个关键字，每个关键字可选择匹配类型
3. 提供正则表达式语法提示和验证
4. 实时显示当前关键字会匹配哪些现有模型

### 6. 数据迁移方案

#### 6.1 数据库迁移脚本
```sql
-- 添加新字段
ALTER TABLE groups ADD COLUMN keywords TEXT DEFAULT '';
ALTER TABLE groups ADD COLUMN match_mode INT DEFAULT 0;

-- 为现有分组设置默认值
UPDATE groups SET keywords = '[]', match_mode = 0 WHERE keywords IS NULL;

-- 添加索引优化查询性能
CREATE INDEX idx_groups_match_mode ON groups(match_mode);
```

#### 6.2 兼容性保证
- 现有分组自动设置为 `GroupMatchModeNameOnly` 模式
- 保持现有 API 接口向后兼容
- 新字段为可选，不影响现有功能

### 7. 测试方案

#### 7.1 单元测试
```go
// internal/matcher/keyword_matcher_test.go
func TestKeywordMatcher_MatchModel(t *testing.T) {
    tests := []struct {
        name      string
        modelName string
        group     model.Group
        expected  bool
    }{
        {
            name:      "精确匹配成功",
            modelName: "gpt-4",
            group: model.Group{
                Keywords:  `[{"pattern":"gpt-4","type":"exact"}]`,
                MatchMode: model.GroupMatchModeKeywordOnly,
            },
            expected: true,
        },
        {
            name:      "正则匹配成功",
            modelName: "claude-3-opus-20240229",
            group: model.Group{
                Keywords:  `[{"pattern":"claude-3-.*","type":"regex"}]`,
                MatchMode: model.GroupMatchModeKeywordOnly,
            },
            expected: true,
        },
        // 更多测试用例...
    }

    matcher := NewKeywordMatcher()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := matcher.MatchModel(tt.modelName, tt.group)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### 7.2 集成测试
- 测试自动分组功能
- 测试模型移除后分组清理
- 测试 API 接口完整性
- 测试数据库事务一致性

#### 7.3 性能测试
- 大量分组和关键字的匹配性能
- 正则表达式编译缓存效果
- 数据库查询优化效果

### 8. 部署方案

#### 8.1 部署步骤
1. **数据库迁移**：执行 SQL 迁移脚本
2. **代码部署**：部署新版本代码
3. **配置验证**：验证新功能正常工作
4. **数据验证**：确认现有数据完整性

#### 8.2 回滚方案
- 保留数据库迁移前的备份
- 代码版本支持快速回滚
- 新增字段设计为可选，不影响回滚

### 9. 监控和日志

#### 9.1 关键指标监控
- 关键字匹配成功率
- 自动分组执行次数
- 模型移除清理成功率
- 正则表达式编译错误率

#### 9.2 日志增强
```go
// 添加详细的匹配日志
log.Infof("keyword matching: model=%s, group=%s, keywords=%s, matched=%v",
    modelName, group.Name, group.Keywords, matched)

// 添加清理操作日志
log.Infof("cleaned up group items: channel=%d, removed_models=%v, affected_groups=%v",
    channelID, removedModels, affectedGroupIDs)
```

### 10. 风险评估和缓解

#### 10.1 主要风险
1. **正则表达式性能风险**
   - 缓解：编译结果缓存，复杂度限制

2. **数据一致性风险**
   - 缓解：事务保护，完整的回滚机制

3. **兼容性风险**
   - 缓解：向后兼容设计，渐进式迁移

#### 10.2 安全考虑
- 正则表达式输入验证和长度限制
- 防止正则表达式拒绝服务攻击
- API 接口权限控制

## 实施计划

### 阶段一：核心功能开发（预计 3-5 天）
1. 数据库结构扩展
2. 关键字匹配引擎开发
3. 模型移除问题修复
4. 单元测试编写

### 阶段二：API 和集成（预计 2-3 天）
1. API 接口扩展
2. 自动分组功能增强
3. 集成测试
4. 性能优化

### 阶段三：前端和部署（预计 2-3 天）
1. 前端界面开发
2. 数据迁移脚本
3. 部署文档
4. 用户文档

### 阶段四：测试和优化（预计 1-2 天）
1. 完整功能测试
2. 性能测试和优化
3. 安全测试
4. 文档完善

## 验收标准

### 功能验收
- ✅ 支持关键字匹配（精确、模糊、正则）
- ✅ 自动分组功能正常工作
- ✅ 模型移除后分组正确清理
- ✅ API 接口完整可用
- ✅ 前端界面友好易用

### 性能验收
- ✅ 关键字匹配响应时间 < 100ms
- ✅ 自动分组处理时间 < 5s
- ✅ 数据库查询优化有效

### 兼容性验收
- ✅ 现有功能完全兼容
- ✅ 数据迁移无损失
- ✅ API 向后兼容

## 总结

本方案通过扩展分组管理的关键字匹配功能，支持正则表达式，并修复模型移除问题，将显著提升 Octopus 系统的自动化管理能力和数据一致性。方案设计充分考虑了兼容性、性能和安全性，确保平滑升级和稳定运行。