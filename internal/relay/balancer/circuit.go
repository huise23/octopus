package balancer

import (
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/utils/log"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota // 正常通行
	StateOpen                       // 熔断中，拒绝所有请求
)

// circuitEntry 单个熔断器条目
type circuitEntry struct {
	State           CircuitState
	Failures        int64           // 总失败次数（成功后清零）
	LastFailureTime time.Time       // 最近一次失败时间
	FailedDays      map[string]bool // 记录发生失败的日期 key: "YYYY-MM-DD"
	PermanentBlock  bool            // 永不重试标记（3天失败后设为true）
	mu              sync.Mutex
}

// 全局熔断器存储
var globalBreaker sync.Map // key: string -> value: *circuitEntry

// circuitKey 生成熔断器键：channelID:channelKeyID:modelName
func circuitKey(channelID, keyID int, modelName string) string {
	return fmt.Sprintf("%d:%d:%s", channelID, keyID, modelName)
}

// getOrCreateEntry 获取或创建熔断器条目
func getOrCreateEntry(key string) *circuitEntry {
	if v, ok := globalBreaker.Load(key); ok {
		return v.(*circuitEntry)
	}
	entry := &circuitEntry{
		State:      StateClosed,
		FailedDays: make(map[string]bool),
	}
	actual, _ := globalBreaker.LoadOrStore(key, entry)
	return actual.(*circuitEntry)
}

// IsTripped 检查通道是否处于熔断状态
// 返回 true 表示该通道应被跳过
func IsTripped(channelID, keyID int, modelName string) bool {
	key := circuitKey(channelID, keyID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return false
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	return entry.State == StateOpen
}

// ShouldRetry 判断熔断的通道是否符合重试条件
// 用于后台探测协程判断是否应该尝试恢复该通道
func ShouldRetry(channelID, keyID int, modelName string) bool {
	key := circuitKey(channelID, keyID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return false
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// 非熔断状态不需要重试
	if entry.State != StateOpen {
		return false
	}

	// 永久封禁不再重试
	if entry.PermanentBlock {
		return false
	}

	now := time.Now()

	// 失败次数>3 且 24小时内不再尝试
	if entry.Failures > 3 && now.Sub(entry.LastFailureTime) < 24*time.Hour {
		return false
	}

	// 时间条件：now - LastFailureTime > 10分钟 * 失败次数
	interval := time.Duration(entry.Failures) * 10 * time.Minute
	if now.Sub(entry.LastFailureTime) <= interval {
		return false
	}

	return true
}

// RecordSuccess 记录成功，重置熔断器状态
func RecordSuccess(channelID, keyID int, modelName string) {
	key := circuitKey(channelID, keyID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.State == StateOpen {
		log.Infof("circuit breaker [%s] Open -> Closed (probe succeeded)", key)
	}

	// 重置全部状态
	entry.State = StateClosed
	entry.Failures = 0
	entry.FailedDays = make(map[string]bool)
	entry.PermanentBlock = false
}

// RecordFailure 记录失败，立即触发熔断
func RecordFailure(channelID, keyID int, modelName string) {
	key := circuitKey(channelID, keyID, modelName)
	entry := getOrCreateEntry(key)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	prevState := entry.State

	// 立即熔断
	entry.State = StateOpen
	entry.LastFailureTime = now
	entry.Failures++

	// 记录失败日期
	if entry.FailedDays == nil {
		entry.FailedDays = make(map[string]bool)
	}
	day := now.Format("2006-01-02")
	entry.FailedDays[day] = true

	// 3天都有失败则永久封禁
	if len(entry.FailedDays) >= 3 {
		entry.PermanentBlock = true
		log.Warnf("circuit breaker [%s] permanently blocked (failed on %d different days)",
			key, len(entry.FailedDays))
	} else if prevState == StateClosed {
		log.Warnf("circuit breaker [%s] Closed -> Open (failure #%d, day %s)",
			key, entry.Failures, day)
	} else {
		log.Warnf("circuit breaker [%s] probe failed (failure #%d, day %s, failed days: %d)",
			key, entry.Failures, day, len(entry.FailedDays))
	}
}

// GetRetryInterval 获取下次可重试的间隔时间（供外部查询）
func GetRetryInterval(channelID, keyID int, modelName string) time.Duration {
	key := circuitKey(channelID, keyID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return 0
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.State != StateOpen {
		return 0
	}

	interval := time.Duration(entry.Failures) * 10 * time.Minute
	elapsed := time.Since(entry.LastFailureTime)
	if elapsed >= interval {
		return 0
	}
	return interval - elapsed
}
