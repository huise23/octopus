# 环境变量同步功能增强规划 - 代理策略规则配置

## 已明确的决策

- 当前环境变量同步功能已实现，代码位于 `internal/services/env_sync.go`
- 当前只支持固定的 "Proxy" 策略（硬编码在第 57 行）
- 需要支持两种策略规则的配置化：不使用代理时的策略和使用代理时的策略
- 使用环境变量方式配置，保持现有架构一致性
- 保持向后兼容性，现有功能不受影响

## 整体规划概述

### 项目目标

为环境变量同步功能增加可配置的代理策略规则，支持：
1. 配置不使用代理时的策略规则（替换 "DIRECT"）
2. 配置使用代理时的策略规则（替换当前硬编码的 "Proxy"）
3. 保持现有功能的完整性和向后兼容性
4. 支持灵活的代理规则配置

### 技术栈

- Go 1.19+
- 现有的环境变量配置机制
- HTTP 客户端用于 API 调用
- 现有的日志系统

### 主要阶段

1. **需求分析和设计阶段** - 分析现有代码，设计新的配置结构
2. **代码实现阶段** - 修改环境同步服务，添加配置项支持
3. **测试验证阶段** - 编写测试用例，验证功能正确性

### 详细任务分解

#### 阶段 1：需求分析和设计（预计 1 小时）

- **任务 1.1**：分析现有代码结构和配置机制
  - 目标：深入理解当前环境同步的实现细节
  - 输入：现有代码 `internal/services/env_sync.go`
  - 输出：代码分析报告，明确修改点
  - 涉及文件：`internal/services/env_sync.go`, `internal/services/env_sync_test.go`
  - 预估工作量：30 分钟

- **任务 1.2**：设计新的环境变量配置项
  - 目标：定义新的环境变量名称和默认值
  - 输入：用户需求和现有配置模式
  - 输出：配置项设计文档
  - 涉及文件：无（设计阶段）
  - 预估工作量：30 分钟

#### 阶段 2：代码实现（预计 1.5 小时）

- **任务 2.1**：修改环境同步服务常量和配置
  - 目标：添加新的环境变量常量定义
  - 输入：设计的配置项
  - 输出：更新的常量定义
  - 涉及文件：`internal/services/env_sync.go`
  - 预估工作量：15 分钟

- **任务 2.2**：修改 EnvSyncService 结构体
  - 目标：添加新的配置字段，支持代理策略规则配置
  - 输入：新的配置项设计
  - 输出：增强的服务结构体
  - 涉及文件：`internal/services/env_sync.go`
  - 预估工作量：15 分钟

- **任务 2.3**：修改 NewEnvSyncService 构造函数
  - 目标：在创建服务实例时读取新的环境变量
  - 输入：环境变量配置
  - 输出：增强的构造函数
  - 涉及文件：`internal/services/env_sync.go`
  - 预估工作量：15 分钟

- **任务 2.4**：重构 SyncDomain 方法
  - 目标：根据使用代理的情况选择不同的策略规则
  - 输入：代理使用标志和配置的策略规则
  - 输出：支持动态策略规则的同步方法
  - 涉及文件：`internal/services/env_sync.go`
  - 预估工作量：30 分钟

- **任务 2.5**：修改调用方传递代理状态
  - 目标：在调用同步服务时传递渠道的代理使用状态
  - 输入：渠道的 Proxy 字段值
  - 输出：更新的调用逻辑
  - 涉及文件：`internal/op/channel.go`
  - 预估工作量：15 分钟

#### 阶段 3：测试验证（预计 1 小时）

- **任务 3.1**：更新现有单元测试
  - 目标：修改现有测试以适应新的 API 变化
  - 输入：修改后的代码
  - 输出：更新的测试用例
  - 涉及文件：`internal/services/env_sync_test.go`
  - 预估工作量：30 分钟

- **任务 3.2**：添加新的测试用例
  - 目标：测试新的代理策略规则配置功能
  - 输入：新的配置项和方法
  - 输出：完整的测试覆盖
  - 涉及文件：`internal/services/env_sync_test.go`
  - 预估工作量：30 分钟

## 需要进一步明确的问题

### 问题 1：环境变量命名约定

当前已有 `OCTOPUS_ENV_SYNC_API_URL`，新增的环境变量应该遵循什么命名模式？

**推荐方案**：

- 方案 A：`OCTOPUS_ENV_SYNC_PROXY_RULE` 和 `OCTOPUS_ENV_SYNC_DIRECT_RULE`
- 方案 B：`OCTOPUS_ENV_SYNC_RULE_PROXY` 和 `OCTOPUS_ENV_SYNC_RULE_DIRECT`

**等待用户选择**：

```
请选择您偏好的方案，或提供其他建议：
[ ] 方案 A：OCTOPUS_ENV_SYNC_PROXY_RULE 和 OCTOPUS_ENV_SYNC_DIRECT_RULE
[ ] 方案 B：OCTOPUS_ENV_SYNC_RULE_PROXY 和 OCTOPUS_ENV_SYNC_RULE_DIRECT
[ ] 其他方案：___________________
```

### 问题 2：默认值策略

当环境变量未设置时，应该使用什么默认值？

**推荐方案**：

- 方案 A：保持向后兼容，代理规则默认为 "Proxy"，直连规则默认为 "DIRECT"
- 方案 B：两个规则都默认为 "Proxy"，保持现有行为
- 方案 C：默认值为空字符串，要求用户必须配置

**等待用户选择**：

```
请选择您偏好的方案：
[ ] 方案 A：代理="Proxy"，直连="DIRECT"（推荐）
[ ] 方案 B：都默认为"Proxy"
[ ] 方案 C：默认为空，强制配置
[ ] 其他方案：___________________
```

### 问题 3：API 方法签名变更

当前 `SyncDomain(domain string)` 方法需要修改为支持代理状态参数。

**推荐方案**：

- 方案 A：修改为 `SyncDomain(domain string, useProxy bool)`
- 方案 B：添加新方法 `SyncDomainWithProxy(domain string, useProxy bool)`，保留原方法
- 方案 C：使用配置对象 `SyncDomain(domain string, config SyncConfig)`

**等待用户选择**：

```
请选择您偏好的方案：
[ ] 方案 A：修改现有方法签名（简洁，但可能影响其他调用方）
[ ] 方案 B：添加新方法，保留原方法（向后兼容）
[ ] 方案 C：使用配置对象（更灵活，便于扩展）
[ ] 其他方案：___________________
```

### 问题 4：是否需要同时支持两种策略

目前分析显示渠道只会在"不使用代理"时才调用同步功能。但为了完整性，是否需要支持"使用代理"时的策略配置？

**推荐方案**：

- 方案 A：只实现不使用代理时的策略配置（简化实现，当前需求）
- 方案 B：同时实现两种策略配置（完整方案，便于扩展）

**等待用户选择**：

```
请选择您偏好的方案：
[ ] 方案 A：仅实现不使用代理时的策略配置
[ ] 方案 B：同时实现两种策略配置（推荐）
[ ] 其他建议：___________________
```

## 用户反馈区域

请在此区域补充您对整体规划的意见和建议：

```
用户补充内容：

---

---

---

```

## 详细技术设计

### 当前代码分析

```go
// 当前第57行的硬编码
payload := fmt.Sprintf("DOMAIN-SUFFIX,%s,Proxy", rootDomain)
```

### 建议的新实现结构

```go
// 新增常量（待用户确认命名）
const (
    envSyncAPIURLKey    = "OCTOPUS_ENV_SYNC_API_URL"
    envSyncProxyRuleKey = "OCTOPUS_ENV_SYNC_PROXY_RULE"     // 使用代理时的规则
    envSyncDirectRuleKey = "OCTOPUS_ENV_SYNC_DIRECT_RULE"   // 不使用代理时的规则
    defaultProxyRule    = "Proxy"                           // 默认代理规则
    defaultDirectRule   = "DIRECT"                          // 默认直连规则
)

// 增强的服务结构
type EnvSyncService struct {
    client     *http.Client
    apiURL     string
    proxyRule  string  // 使用代理时的策略规则
    directRule string  // 不使用代理时的策略规则
}

// 增强的构造函数
func NewEnvSyncService() *EnvSyncService {
    proxyRule := os.Getenv(envSyncProxyRuleKey)
    if proxyRule == "" {
        proxyRule = defaultProxyRule
    }

    directRule := os.Getenv(envSyncDirectRuleKey)
    if directRule == "" {
        directRule = defaultDirectRule
    }

    return &EnvSyncService{
        client: &http.Client{
            Timeout: requestTimeout,
        },
        apiURL:     os.Getenv(envSyncAPIURLKey),
        proxyRule:  proxyRule,
        directRule: directRule,
    }
}

// 修改后的同步方法（方案待确认）
func (s *EnvSyncService) SyncDomain(domain string, useProxy bool) error {
    // ... 现有逻辑 ...

    // 根据代理状态选择规则
    var rule string
    if useProxy {
        rule = s.proxyRule
    } else {
        rule = s.directRule
    }

    // 构建请求体
    payload := fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", rootDomain, rule)

    // ... 其余逻辑保持不变 ...
}
```

### 调用方修改

```go
// internal/op/channel.go 中的修改
// 当前调用
envSyncService.SyncDomainAsync(channel.BaseURL)

// 修改后的调用（传递代理状态）
envSyncService.SyncDomainAsyncWithProxy(channel.BaseURL, channel.Proxy)
```

### 测试用例设计

```go
func TestSyncDomainWithProxyRules(t *testing.T) {
    tests := []struct {
        name           string
        domain         string
        useProxy       bool
        proxyRule      string
        directRule     string
        expectedPayload string
    }{
        {
            name:           "使用代理时的自定义规则",
            domain:         "api.example.com",
            useProxy:       true,
            proxyRule:      "MyProxy",
            directRule:     "MyDirect",
            expectedPayload: "DOMAIN-SUFFIX,example.com,MyProxy",
        },
        {
            name:           "不使用代理时的自定义规则",
            domain:         "api.example.com",
            useProxy:       false,
            proxyRule:      "MyProxy",
            directRule:     "MyDirect",
            expectedPayload: "DOMAIN-SUFFIX,example.com,MyDirect",
        },
        {
            name:           "默认规则测试",
            domain:         "api.example.com",
            useProxy:       true,
            proxyRule:      "", // 使用默认值
            directRule:     "", // 使用默认值
            expectedPayload: "DOMAIN-SUFFIX,example.com,Proxy",
        },
    }

    // 测试实现...
}
```

## 风险评估和缓解策略

### 技术风险

1. **向后兼容性风险**
   - 风险：修改方法签名可能影响其他调用方
   - 缓解：采用新方法 + 保留原方法的策略

2. **配置错误风险**
   - 风险：用户配置错误的策略规则导致同步失败
   - 缓解：提供合理的默认值，添加配置验证

3. **测试覆盖率风险**
   - 风险：新功能测试不充分
   - 缓解：编写全面的单元测试和集成测试

### 实施建议

1. **分阶段实施**：先实现核心功能，再完善边缘情况
2. **充分测试**：确保所有场景都有测试覆盖
3. **文档更新**：更新相关文档说明新的配置项
4. **监控验证**：部署后监控新功能的运行状况

## 验收标准

- [ ] 支持通过环境变量配置代理策略规则
- [ ] 支持通过环境变量配置直连策略规则
- [ ] 现有功能保持完全兼容
- [ ] 新增功能有完整的单元测试覆盖
- [ ] 配置项有合理的默认值
- [ ] 错误处理和日志记录完善
- [ ] 代码通过现有的代码质量检查
- [ ] 集成测试验证完整流程正确性