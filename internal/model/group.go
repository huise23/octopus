package model

type GroupMode int

const (
	GroupModeRoundRobin GroupMode = 1 // 轮询：依次循环选择渠道
	GroupModeRandom     GroupMode = 2 // 随机：每次随机选择一个渠道
	GroupModeFailover   GroupMode = 3 // 故障转移：按优先级选择，失败时降级到下一个
	GroupModeWeighted   GroupMode = 4 // 加权分配：按优权重分配流量
)

// GroupMatchMode 分组匹配模式
type GroupMatchMode int

const (
	GroupMatchModeNameOnly    GroupMatchMode = 0 // 仅分组名称匹配（默认，保持兼容）
	GroupMatchModeKeywordOnly GroupMatchMode = 1 // 仅关键字匹配
	GroupMatchModeBoth        GroupMatchMode = 2 // 分组名称 + 关键字匹配
)

// GroupKeyword 关键字配置结构
type GroupKeyword struct {
	Pattern string `json:"pattern"` // 匹配模式（支持正则）
	Type    string `json:"type"`    // 匹配类型：exact, fuzzy, regex
}

type Group struct {
	ID        int            `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"unique;not null"`
	Mode      GroupMode      `json:"mode" gorm:"not null"`
	Keywords  string         `json:"keywords" gorm:"type:text;default:''"`     // JSON 格式的关键字列表
	MatchMode GroupMatchMode `json:"match_mode" gorm:"default:0"`              // 匹配模式
	Items     []GroupItem    `json:"items,omitempty" gorm:"foreignKey:GroupID"`
}

type GroupItem struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	GroupID   int    `json:"group_id" gorm:"not null;index:idx_group_channel_model,unique"` // 创建时不携带此字段,更新时需要
	ChannelID int    `json:"channel_id" gorm:"not null;index:idx_group_channel_model,unique"`
	ModelName string `json:"model_name" gorm:"not null;index:idx_group_channel_model,unique"`
	Priority  int    `json:"priority"`
	Weight    int    `json:"weight"`
}

// GroupUpdateRequest 分组更新请求 - 仅包含变更的数据
type GroupUpdateRequest struct {
	ID            int                      `json:"id" binding:"required"`
	Name          *string                  `json:"name,omitempty"`            // 仅在名称变更时发送
	Mode          *GroupMode               `json:"mode,omitempty"`            // 仅在模式变更时发送
	Keywords      *string                  `json:"keywords,omitempty"`        // 仅在关键字变更时发送
	MatchMode     *GroupMatchMode          `json:"match_mode,omitempty"`      // 仅在匹配模式变更时发送
	ItemsToAdd    []GroupItemAddRequest    `json:"items_to_add,omitempty"`    // 新增的 items
	ItemsToUpdate []GroupItemUpdateRequest `json:"items_to_update,omitempty"` // 更新的 items (priority 变更)
	ItemsToDelete []int                    `json:"items_to_delete,omitempty"` // 删除的 item IDs
}

// GroupItemAddRequest 新增 item 请求
type GroupItemAddRequest struct {
	ChannelID int    `json:"channel_id" binding:"required"`
	ModelName string `json:"model_name" binding:"required"`
	Priority  int    `json:"priority,omitempty"`
	Weight    int    `json:"weight,omitempty"`
}

// GroupItemUpdateRequest 更新 item 请求
type GroupItemUpdateRequest struct {
	ID       int `json:"id" binding:"required"`
	Priority int `json:"priority,omitempty"`
	Weight   int `json:"weight,omitempty"`
}
type GroupIDAndLLMName struct {
	ChannelID int
	ModelName string
}
