package balancer

import (
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
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
	State             CircuitState
	Failures          int64           // 总失败次数（成功后清零）
	RateLimitFailures int64           // 429 失败计数（累计到3才计入失败）
	LastFailureTime   time.Time       // 最近一次失败时间
	FailedDays        map[string]bool // 记录发生失败的日期 key: "YYYY-MM-DD"
	PermanentBlock    bool            // 永不重试标记（3天失败后设为true）
	mu                sync.Mutex
}

// 全局熔断器存储
var globalBreaker sync.Map // key: string -> value: *circuitEntry

// RestoreFromDB restores circuit breaker state from persistent storage.
func RestoreFromDB(states []model.CircuitBreakerState) {
	for _, state := range states {
		key := circuitKey(state.ChannelID, state.ChannelKeyID, state.ModelName)
		entry := getOrCreateEntry(key)
		entry.mu.Lock()
		entry.State = CircuitState(state.State)
		entry.Failures = state.Failures
		entry.RateLimitFailures = state.RateLimitFailures
		entry.LastFailureTime = state.LastFailureTime
		if state.FailedDays == nil {
			entry.FailedDays = make(map[string]bool)
		} else {
			entry.FailedDays = state.FailedDays
		}
		entry.PermanentBlock = state.PermanentBlock
		entry.mu.Unlock()
	}
}

// GetStateSnapshot returns a copy of circuit breaker state for persistence.
func GetStateSnapshot(channelID, keyID int, modelName string) (model.CircuitBreakerState, bool) {
	key := circuitKey(channelID, keyID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return model.CircuitBreakerState{}, false
	}
	entry := v.(*circuitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	failedDays := make(map[string]bool, len(entry.FailedDays))
	for k, v := range entry.FailedDays {
		failedDays[k] = v
	}
	return model.CircuitBreakerState{
		ChannelID:         channelID,
		ChannelKeyID:      keyID,
		ModelName:         modelName,
		State:             int(entry.State),
		Failures:          entry.Failures,
		RateLimitFailures: entry.RateLimitFailures,
		LastFailureTime:   entry.LastFailureTime,
		FailedDays:        failedDays,
		PermanentBlock:    entry.PermanentBlock,
	}, true
}

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
	entry.RateLimitFailures = 0
	entry.FailedDays = make(map[string]bool)
	entry.PermanentBlock = false
}

// RecordFailure 记录失败，立即触发熔断

func RecordFailure(channelID, keyID int, modelName string) {
	RecordFailureWithStatus(channelID, keyID, modelName, 0)
}

// RecordFailureWithStatus 记录失败，可根据 statusCode 做特殊处理
// 对 429：累计 3 次才计为一次失败
func RecordFailureWithStatus(channelID, keyID int, modelName string, statusCode int) {
	key := circuitKey(channelID, keyID, modelName)
	entry := getOrCreateEntry(key)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	prevState := entry.State

	if statusCode == 429 {
		entry.RateLimitFailures++
		if entry.RateLimitFailures < 3 {
			log.Warnf("circuit breaker [%s] rate limited (429 #%d/3)", key, entry.RateLimitFailures)
			return
		}
		// 达到 3 次，计为一次失败并清零
		entry.RateLimitFailures = 0
	}

	// 立即熔断（或探测失败）
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
