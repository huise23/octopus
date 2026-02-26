package balancer

import (
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

// Iterator 统一的负载均衡迭代器
// 内部编排：策略排序 + 粘性优先 + 决策追踪
type Iterator struct {
	candidates []model.GroupItem
	index      int
	stickyIdx  int    // 粘性通道在 candidates 中的位置，-1 表示无
	modelName  string // 请求模型名（用于熔断检查）

	// 内嵌追踪
	attempts []model.ChannelAttempt
	count    int
}

// NewIterator 创建负载均衡迭代器
// 自动处理：策略排序 + 粘性通道提前
func NewIterator(group model.Group, apiKeyID int, requestModel string, stickyTTL time.Duration) *Iterator {
	// 直接使用 group.Items 作为候选列表
	candidates := make([]model.GroupItem, len(group.Items))
	copy(candidates, group.Items)

	stickyIdx := -1
	if stickyTTL > 0 {
		if sticky := GetSticky(apiKeyID, requestModel, stickyTTL); sticky != nil {
			for i, item := range candidates {
				if item.ChannelID == sticky.ChannelID {
					if i > 0 {
						// 将粘性通道移到最前面
						stickyItem := candidates[i]
						copy(candidates[1:i+1], candidates[0:i])
						candidates[0] = stickyItem
					}
					stickyIdx = 0
					break
				}
			}
		}
	}

	return &Iterator{
		candidates: candidates,
		index:      -1,
		stickyIdx:  stickyIdx,
		modelName:  requestModel,
	}
}

// Next 移动到下一个候选，返回 false 表示遍历完成
func (it *Iterator) Next() bool {
	it.index++
	return it.index < len(it.candidates)
}

// Item 返回当前候选的 GroupItem
func (it *Iterator) Item() model.GroupItem {
	return it.candidates[it.index]
}

// IsSticky 当前候选是否为粘性通道
func (it *Iterator) IsSticky() bool {
	return it.stickyIdx >= 0 && it.index == it.stickyIdx
}

// Len 返回候选列表长度
func (it *Iterator) Len() int {
	return len(it.candidates)
}

// Index 返回当前迭代位置（0-based）
func (it *Iterator) Index() int {
	return it.index
}

// Skip 记录当前通道被跳过（通道禁用、无Key、类型不兼容等）
func (it *Iterator) Skip(channelID int, channelName, msg string) {
	it.count++
	it.attempts = append(it.attempts, model.ChannelAttempt{
		ChannelID:   channelID,
		ChannelName: channelName,
		ModelName:   it.candidates[it.index].ModelName,
		AttemptNum:  it.count,
		Success:     false,
		Error:       msg,
	})
}

// SkipCircuitBreak 检查熔断状态，若已熔断自动记录并返回 true
// keyID 参数用于熔断检查，传 0 表示不区分 key
func (it *Iterator) SkipCircuitBreak(channelID, keyID int, channelName string) bool {
	modelName := it.candidates[it.index].ModelName
	tripped := IsTripped(channelID, keyID, modelName)
	if !tripped {
		return false
	}
	it.count++
	it.attempts = append(it.attempts, model.ChannelAttempt{
		ChannelID:   channelID,
		ChannelName: channelName,
		ModelName:   modelName,
		AttemptNum:  it.count,
		Success:     false,
		Error:       "circuit_break",
	})
	return true
}

// StartAttempt 开始一次真实转发尝试，返回 Span 用于记录结果
func (it *Iterator) StartAttempt(channelID int, channelName string) *AttemptSpan {
	it.count++
	return &AttemptSpan{
		attempt: model.ChannelAttempt{
			ChannelID:   channelID,
			ChannelName: channelName,
			ModelName:   it.candidates[it.index].ModelName,
			AttemptNum:  it.count,
		},
		startTime: time.Now(),
		iter:      it,
	}
}

// Attempts 返回所有决策记录（交给日志模块持久化）
func (it *Iterator) Attempts() []model.ChannelAttempt {
	return it.attempts
}

// AttemptSpan 管理单次通道尝试的生命周期（计时、状态、结果）
type AttemptSpan struct {
	attempt   model.ChannelAttempt
	startTime time.Time
	iter      *Iterator
	ended     bool
}

// End 结束尝试：设置状态，自动计算耗时，追加到 Iterator
func (s *AttemptSpan) End(success bool, msg string) {
	if s.ended {
		return
	}
	s.ended = true
	s.attempt.Success = success
	s.attempt.Error = msg
	s.attempt.Duration = int(time.Since(s.startTime).Milliseconds())
	s.iter.attempts = append(s.iter.attempts, s.attempt)
}

// Duration 返回从开始到现在的耗时
func (s *AttemptSpan) Duration() time.Duration {
	return time.Since(s.startTime)
}
