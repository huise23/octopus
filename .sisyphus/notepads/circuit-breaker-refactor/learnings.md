# 熔断策略重构学习笔记

## 2025-02-26 需求分析

### 核心变更

1. **失败一次即不可用** - 不再需要连续失败阈值（原来是5次）
2. **动态恢复探测条件**: `now - LastFailureTime > 10分钟 * 失败次数`
3. **探测参数**: 使用当前请求参数，截取前1K字符减少token消耗
4. **Token统计**: 按原规则统计，用单独字段存储(ProbeInputTokens, ProbeOutputTokens)
5. **并发控制**: 同一 channel 同时只启动一个重试协程（避免rpm限制）
6. **超时**: 重试超时30秒
7. **失败限制**:
   - 失败次数>3 且 同一天内：不再尝试
   - 连续3天都有失败（FailedDays达到3个不同日期）：**永不重试**
8. **成功后**: 清零失败次数、失败天数，标记可用

### 关键代码位置

- circuit.go: 熔断状态管理 (circuitEntry, IsTripped, RecordSuccess, RecordFailure)
- iterator.go: SkipCircuitBreak() 调用 IsTripped()
- relay.go: Handler() 主循环，attempt() 记录成功/失败
- metrics.go: Token统计和日志保存

### 数据结构变更

**旧 circuitEntry**:
- State, ConsecutiveFailures, LastFailureTime, TripCount

**新 circuitEntry**:
- State (只用 StateClosed / StateOpen)
- Failures (总失败次数)
- LastFailureTime
- FailedDays map[string]bool (记录失败日期)
- PermanentBlock bool (3天失败后永不重试)

### 新增功能

- probe.go: Channel级别重试协程管理
  - channelRetryRunning sync.Map (channelID -> bool)
  - ShouldRetry() 判断是否符合重试条件
  - TriggerProbe() 启动重试协程
  - buildProbeRequest() 截取前1K字符
