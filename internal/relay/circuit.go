package relay

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrorType 错误类型分级
type ErrorType int

const (
	// ErrorTypeUnknown 未知错误类型
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeRateLimit 速率限制 (429) - 短期冷却 5 分钟
	ErrorTypeRateLimit
	// ErrorTypeServerError 服务端错误 (5xx) - 中期熔断 10 分钟
	ErrorTypeServerError
	// ErrorTypeAuthError 认证错误 (401/403) - 长期禁用
	ErrorTypeAuthError
	// ErrorTypeTimeout 超时错误 - 降低优先级但不熔断
	ErrorTypeTimeout
	// ErrorTypeNetworkError 网络错误 - 短期重试
	ErrorTypeNetworkError
)

// FailureRecord 失败记录
type FailureRecord struct {
	Timestamp  time.Time
	StatusCode int
	ErrorType  ErrorType
}

// CircuitBreaker 熔断器 - 针对 group 下的每个 item (channel + model)
type CircuitBreaker struct {
	mu sync.RWMutex

	// 存储每个 item 的失败记录: key = groupID:channelID:modelName
	failures map[string][]FailureRecord

	// 配置
	failureThreshold      int           // 连续失败多少次后熔断
	windowDuration        time.Duration // 时间窗口 (10分钟)
	rateLimitCooldown     time.Duration // 429 冷却时间 (5分钟)
	serverErrorCooldown   time.Duration // 5xx 冷却时间 (10分钟)
	authErrorCooldown     time.Duration // 4xx 认证错误冷却 (15分钟)
	networkErrorCooldown  time.Duration // 网络错误冷却 (2分钟)
	timeoutCooldown       time.Duration // 超时错误冷却 (2分钟)
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		failures:           make(map[string][]FailureRecord),
		failureThreshold:   2,
		windowDuration:     10 * time.Minute,
		rateLimitCooldown:  5 * time.Minute,
		serverErrorCooldown: 10 * time.Minute,
		authErrorCooldown:   15 * time.Minute,
		networkErrorCooldown: 2 * time.Minute,
		timeoutCooldown:     2 * time.Minute,
	}
}

// globalCircuitBreaker 全局熔断器实例
var globalCircuitBreaker = NewCircuitBreaker()

// GetCircuitBreaker 获取全局熔断器
func GetCircuitBreaker() *CircuitBreaker {
	return globalCircuitBreaker
}

// makeKey 生成 item 的唯一 key
func (cb *CircuitBreaker) makeKey(groupID, channelID int, modelName string) string {
	return formatKey(groupID, channelID, modelName)
}

// ClassifyError 根据错误信息和状态码分类错误类型
func ClassifyError(err error, statusCode int, responseBody string) ErrorType {
	if err == nil {
		// 没有错误，根据状态码判断
		if statusCode >= 200 && statusCode < 300 {
			return ErrorTypeUnknown // 表示没有错误
		}
	}

	// 首先根据状态码判断
	if statusCode != 0 {
		switch {
		case statusCode == 429:
			return ErrorTypeRateLimit
		case statusCode >= 500 && statusCode < 600:
			return ErrorTypeServerError
		case statusCode == 401 || statusCode == 403:
			return ErrorTypeAuthError
		}
	}

	// 根据错误信息判断
	if err != nil {
		errStr := strings.ToLower(err.Error())

		// 检查是否为超时错误
		if strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "deadline exceeded") ||
			strings.Contains(errStr, "context deadline") ||
			strings.Contains(errStr, "timed out") {
			return ErrorTypeTimeout
		}

		// 检查是否为网络错误
		if strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "broken pipe") ||
			strings.Contains(errStr, "network") ||
			strings.Contains(errStr, "dns") ||
			strings.Contains(errStr, "no such host") ||
			strings.Contains(errStr, "tcp") {
			return ErrorTypeNetworkError
		}

		// 检查是否为 URL 解析错误
		if _, urlErr := err.(*url.Error); urlErr {
			return ErrorTypeNetworkError
		}

		// 检查是否为 net 包错误
		if _, netErr := err.(net.Error); netErr {
			timeoutErr, ok := err.(interface{ Timeout() bool })
			if ok && timeoutErr.Timeout() {
				return ErrorTypeTimeout
			}
			temporaryErr, ok := err.(interface{ Temporary() bool })
			if ok && temporaryErr.Temporary() {
				return ErrorTypeNetworkError
			}
			return ErrorTypeNetworkError
		}
	}

	// 根据 response body 判断
	if responseBody != "" {
		bodyStr := strings.ToLower(responseBody)
		if strings.Contains(bodyStr, "rate limit") ||
			strings.Contains(bodyStr, "too many requests") ||
			strings.Contains(bodyStr, "quota exceeded") {
			return ErrorTypeRateLimit
		}
		if strings.Contains(bodyStr, "unauthorized") ||
			strings.Contains(bodyStr, "forbidden") ||
			strings.Contains(bodyStr, "invalid api key") ||
			strings.Contains(bodyStr, "authentication") {
			return ErrorTypeAuthError
		}
		if strings.Contains(bodyStr, "internal server error") ||
			strings.Contains(bodyStr, "service unavailable") ||
			strings.Contains(bodyStr, "500") ||
			strings.Contains(bodyStr, "502") ||
			strings.Contains(bodyStr, "503") ||
			strings.Contains(bodyStr, "504") {
			return ErrorTypeServerError
		}
	}

	// 默认返回未知错误类型
	return ErrorTypeUnknown
}

// GetCooldownTime 获取错误类型对应的冷却时间
func (cb *CircuitBreaker) GetCooldownTime(errType ErrorType) time.Duration {
	switch errType {
	case ErrorTypeRateLimit:
		return cb.rateLimitCooldown
	case ErrorTypeServerError:
		return cb.serverErrorCooldown
	case ErrorTypeAuthError:
		return cb.authErrorCooldown
	case ErrorTypeTimeout:
		return cb.timeoutCooldown
	case ErrorTypeNetworkError:
		return cb.networkErrorCooldown
	default:
		return 0
	}
}

// GetErrorTypeName 获取错误类型名称
func GetErrorTypeName(errType ErrorType) string {
	switch errType {
	case ErrorTypeRateLimit:
		return "RateLimit"
	case ErrorTypeServerError:
		return "ServerError"
	case ErrorTypeAuthError:
		return "AuthError"
	case ErrorTypeTimeout:
		return "Timeout"
	case ErrorTypeNetworkError:
		return "NetworkError"
	default:
		return "Unknown"
	}
}

// IsRecoverable 判断错误是否可恢复
func IsRecoverable(errType ErrorType) bool {
	switch errType {
	case ErrorTypeRateLimit, ErrorTypeTimeout, ErrorTypeNetworkError, ErrorTypeUnknown:
		return true
	case ErrorTypeServerError:
		return true
	case ErrorTypeAuthError:
		return false
	default:
		return true
	}
}

// cleanupExpired 清理过期的失败记录
func (cb *CircuitBreaker) cleanupExpired(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	records, exists := cb.failures[key]
	if !exists {
		return
	}

	now := time.Now()
	validRecords := make([]FailureRecord, 0, len(records))
	for _, record := range records {
		if now.Sub(record.Timestamp) < cb.windowDuration {
			validRecords = append(validRecords, record)
		}
	}

	if len(validRecords) == 0 {
		delete(cb.failures, key)
	} else {
		cb.failures[key] = validRecords
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure(groupID, channelID int, modelName string, statusCode int, errType ErrorType) {
	key := cb.makeKey(groupID, channelID, modelName)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	record := FailureRecord{
		Timestamp:  now,
		StatusCode: statusCode,
		ErrorType:  errType,
	}

	// 先清理过期记录
	cb.failures[key] = append(cb.failures[key], record)

	// 清理超出时间窗口的记录
	validRecords := make([]FailureRecord, 0, len(cb.failures[key]))
	for _, r := range cb.failures[key] {
		if now.Sub(r.Timestamp) < cb.windowDuration {
			validRecords = append(validRecords, r)
		}
	}
	if len(validRecords) == 0 {
		delete(cb.failures, key)
	} else {
		cb.failures[key] = validRecords
	}
}

// GetRecentFailures 获取最近的失败记录 (在时间窗口内)
func (cb *CircuitBreaker) GetRecentFailures(groupID, channelID int, modelName string) []FailureRecord {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	key := cb.makeKey(groupID, channelID, modelName)
	records, exists := cb.failures[key]
	if !exists {
		return nil
	}

	now := time.Now()
	recent := make([]FailureRecord, 0)
	for _, record := range records {
		if now.Sub(record.Timestamp) < cb.windowDuration {
			recent = append(recent, record)
		}
	}

	return recent
}

// ShouldSkip 判断是否应该跳过该 item
// 返回 (shouldSkip, reason, errorType)
func (cb *CircuitBreaker) ShouldSkip(groupID, channelID int, modelName string) (bool, string, ErrorType) {
	key := cb.makeKey(groupID, channelID, modelName)

	cb.mu.RLock()
	records, exists := cb.failures[key]
	cb.mu.RUnlock()

	if !exists {
		return false, "", ErrorTypeUnknown
	}

	now := time.Now()
	count := 0
	var lastError ErrorType
	var lastTime time.Time

	// 统计时间窗口内的失败次数
	for _, record := range records {
		if now.Sub(record.Timestamp) < cb.windowDuration {
			count++
			lastError = record.ErrorType
			if record.Timestamp.After(lastTime) {
				lastTime = record.Timestamp
			}
		}
	}

	// 如果失败次数不足阈值，不跳过
	if count < cb.failureThreshold {
		return false, "", ErrorTypeUnknown
	}

	// 根据最后一次错误的类型判断冷却时间
	switch lastError {
	case ErrorTypeRateLimit:
		if now.Sub(lastTime) < cb.rateLimitCooldown {
			return true, "rate limited, cooling down", ErrorTypeRateLimit
		}
	case ErrorTypeServerError:
		if now.Sub(lastTime) < cb.serverErrorCooldown {
			return true, "server error, cooling down", ErrorTypeServerError
		}
	case ErrorTypeAuthError:
		// 认证错误需要人工干预，暂时跳过更长时间
		if now.Sub(lastTime) < cb.authErrorCooldown {
			return true, "auth error, cooling down", ErrorTypeAuthError
		}
	case ErrorTypeTimeout:
		if now.Sub(lastTime) < cb.timeoutCooldown {
			return true, "timeout, cooling down", ErrorTypeTimeout
		}
	case ErrorTypeNetworkError:
		if now.Sub(lastTime) < cb.networkErrorCooldown {
			return true, "network error, cooling down", ErrorTypeNetworkError
		}
	}

	return false, "", ErrorTypeUnknown
}

// GetFailureCount 获取失败的次数
func (cb *CircuitBreaker) GetFailureCount(groupID, channelID int, modelName string) int {
	records := cb.GetRecentFailures(groupID, channelID, modelName)
	return len(records)
}

// Clear 清除指定 item 的失败记录 (用于手动重置或测试)
func (cb *CircuitBreaker) Clear(groupID, channelID int, modelName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	key := cb.makeKey(groupID, channelID, modelName)
	delete(cb.failures, key)
}

// formatKey 格式化 key
func formatKey(groupID, channelID int, modelName string) string {
	return fmt.Sprintf("%d:%d:%s", groupID, channelID, modelName)
}
