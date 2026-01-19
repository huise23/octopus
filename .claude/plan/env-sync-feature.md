# 环境变量同步功能实施规划

## 1. 功能目标和验收标准

### 功能目标
实现渠道保存时自动同步根域名到环境变量的功能，当渠道未勾选"使用代理"时，自动调用外部 API 将渠道的根域名添加到代理规则中。

### 验收标准
- [ ] 保存渠道时，若未勾选"使用代理"，自动提取根域名
- [ ] 成功调用外部 API 更新代理规则
- [ ] API 调用失败时记录日志但不影响渠道保存
- [ ] 支持各种域名格式的正确提取（包括子域名、多级域名等）
- [ ] 配置文件可正确设置 API URL
- [ ] API 调用超时时间为 5 秒
- [ ] 失败时不重试，直接记录错误日志

## 2. 技术架构设计

### 架构图
```
┌─────────────────┐
│   前端界面      │
│  (Next.js)      │
└────────┬────────┘
         │ HTTP请求
┌────────▼────────┐
│   渠道处理器     │
│ (channel.go)    │
└────────┬────────┘
         │ 调用
┌────────▼────────┐
│   渠道业务逻辑   │
│  (channel.go)   │
└────────┬────────┘
         │ 触发
┌────────▼────────┐
│  环境同步服务    │
│ (env_sync.go)   │
└───────┬──────────┘
        │
┌───────▼───────┐
│ 域名提取工具   │
│ (domain.go)   │
└───────┬───────┘
        │
┌───────▼───────┐
│ 外部 API 调用 │
│ 超时: 5秒     │
│ 重试: 不重试  │
└───────────────┘
```

### 核心组件
1. **域名提取器**：使用 Public Suffix List 提取根域名
2. **API 调用服务**：封装外部 API 调用逻辑，5秒超时，不重试
3. **配置管理**：管理 API URL 等配置项
4. **日志记录**：记录同步操作的详细信息

### 技术决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| API 认证 | 无需认证 | 直接调用外部 API |
| 错误处理 | 仅记录日志 | API 失败不影响渠道保存 |
| 配置方式 | 后端配置文件 | API URL 在配置文件中设置 |
| 域名提取 | Public Suffix List | 使用标准库提取根域名 |
| 超时设置 | 5 秒 | 平衡响应速度和成功率 |
| 重试策略 | 不重试 | 快速失败，避免阻塞 |

## 3. 详细实施步骤

### 步骤 1：环境准备（优先级：高）

**任务 1.1：安装 Public Suffix List 依赖**
```bash
go get golang.org/x/net/publicsuffix
```
- 涉及文件：`go.mod`
- 预估时间：0.5 小时

**任务 1.2：添加 API URL 配置项**
- 在配置文件中添加外部 API 的 URL 配置
- 涉及文件：
  - `configs/config.yaml`
  - `internal/config/config.go`
- 预估时间：0.5 小时

### 步骤 2：实现域名提取功能（优先级：高）

**任务 2.1：创建域名提取工具**
- 创建 `internal/utils/domain.go`
- 实现提取根域名的函数 `ExtractRootDomain(domain string) (string, error)`
- 涉及文件：`internal/utils/domain.go`（新建）
- 预估时间：1 小时

**任务 2.2：编写域名提取测试**
- 测试各种域名格式
- 涉及文件：`internal/utils/domain_test.go`（新建）
- 预估时间：1 小时

### 步骤 3：实现 API 调用服务（优先级：高）

**任务 3.1：创建环境同步服务**
- 创建 `internal/services/env_sync.go`
- 实现 HTTP 客户端和 API 调用逻辑
- 添加 5 秒超时设置
- 添加错误处理和日志记录
- 涉及文件：`internal/services/env_sync.go`（新建）
- 预估时间：1.5 小时

**任务 3.2：编写 API 调用测试**
- 测试成功调用场景
- 测试网络错误场景
- 测试超时场景
- 涉及文件：`internal/services/env_sync_test.go`（新建）
- 预估时间：1 小时

### 步骤 4：集成到渠道保存流程（优先级：高）

**任务 4.1：修改渠道保存逻辑**
- 修改 `internal/op/channel.go` 中的保存逻辑
- 在保存成功后检查"使用代理"选项
- 如果未勾选，触发域名同步
- 确保异步执行，不影响响应时间
- 涉及文件：`internal/op/channel.go`
- 预估时间：1 小时

**任务 4.2：添加日志记录**
- 记录同步操作的开始和结果
- 记录错误详情但不影响业务流程
- 涉及文件：`internal/services/env_sync.go`
- 预估时间：0.5 小时

### 步骤 5：测试验证（优先级：中）

**任务 5.1：集成测试**
- 测试完整的渠道保存流程
- 验证勾选/不勾选"使用代理"的行为差异
- 涉及文件：`tests/integration/channel_test.go`（新建）
- 预估时间：2 小时

**任务 5.2：边界情况测试**
- 测试无效域名
- 测试空字符串
- 测试特殊字符
- 预估时间：1 小时

## 4. 代码文件清单

### 新建文件

| 文件路径 | 说明 | 优先级 |
|----------|------|--------|
| `internal/utils/domain.go` | 域名提取工具 | 高 |
| `internal/utils/domain_test.go` | 域名提取测试 | 高 |
| `internal/services/env_sync.go` | 环境同步服务 | 高 |
| `internal/services/env_sync_test.go` | 环境同步测试 | 高 |
| `tests/integration/channel_test.go` | 集成测试 | 中 |

### 修改文件

| 文件路径 | 修改内容 | 优先级 |
|----------|----------|--------|
| `go.mod` | 添加 Public Suffix List 依赖 | 高 |
| `go.sum` | 更新依赖校验和 | 高 |
| `configs/config.yaml` | 添加 API URL 配置项 | 高 |
| `internal/config/config.go` | 添加配置结构定义 | 高 |
| `internal/op/channel.go` | 集成同步逻辑 | 高 |

## 5. 测试计划

### 单元测试 - 域名提取

**测试用例：**
```go
// 普通域名
example.com → example.com

// 子域名
api.example.com → example.com
v1.api.example.com → example.com

// 多级域名
api.co.uk → co.uk

// 边界情况
"" → error
"invalid" → error
"http://example.com" → example.com
"https://api.example.com/path" → example.com
```

### 单元测试 - API 调用

**测试用例：**
- 成功调用：返回 200 状态码
- 网络错误：连接被拒绝
- 超时错误：5 秒后超时
- HTTP 错误：返回 4xx/5xx 状态码

### 集成测试

**测试场景：**
1. 保存渠道（未勾选代理）→ 触发 API 调用
2. 保存渠道（勾选代理）→ 不触发 API 调用
3. 保存渠道（API 失败）→ 渠道保存成功，记录错误日志
4. 批量保存多个渠道 → 正确处理并发

### 性能测试

- API 调用对保存操作的性能影响
- 并发场景下的表现
- 内存占用情况

## 6. 风险和注意事项

### 技术风险

**风险 1：Public Suffix List 更新**
- 描述：本地缓存的列表可能过时
- 概率：低
- 影响：中
- 缓解措施：
  - 定期更新依赖版本
  - 考虑使用在线 API 作为备选方案

**风险 2：API 可用性**
- 描述：外部 API 可能不可用
- 概率：中
- 影响：低（已设计为不影响主流程）
- 缓解措施：
  - 设置 5 秒超时避免长时间阻塞
  - 记录详细日志便于排查
  - 建议添加监控告警

**风险 3：性能影响**
- 描述：API 调用可能增加保存时间
- 概率：低
- 影响：低（使用异步调用）
- 缓解措施：
  - 使用 goroutine 异步执行
  - 设置合理超时时间
  - 监控 API 响应时间

### 业务风险

**风险 1：域名提取错误**
- 描述：错误的域名可能影响代理规则
- 概率：低
- 影响：高
- 缓解措施：
  - 充分测试各种域名格式
  - 使用标准的 Public Suffix List
  - 添加输入验证

**风险 2：重复同步**
- 描述：可能产生重复的代理规则
- 概率：中
- 影响：中
- 缓解措施：
  - 外部 API 应支持幂等性
  - 考虑添加去重逻辑

### 注意事项

1. **配置安全**
   - API URL 配置应使用环境变量或加密配置
   - 避免在代码中硬编码敏感信息

2. **日志规范**
   - 日志记录不包含敏感信息
   - 使用结构化日志便于分析
   - 记录足够的信息用于问题排查

3. **监控告警**
   - 监控 API 调用的成功率
   - 监控 API 响应延迟
   - 设置失败率告警

4. **错误处理**
   - 所有错误都应记录日志
   - 错误信息应清晰明确
   - 不影响主业务流程

## 7. 实施时间表

| 阶段 | 任务 | 预计时间 | 依赖 |
|------|------|----------|------|
| 1 | 环境准备 | 1 小时 | 无 |
| 2 | 域名提取功能 | 2 小时 | 阶段 1 |
| 3 | API 调用服务 | 2.5 小时 | 阶段 1 |
| 4 | 集成到渠道保存 | 1.5 小时 | 阶段 2, 3 |
| 5 | 测试验证 | 3 小时 | 阶段 4 |
| **总计** | | **10 小时** | |

## 8. 后续优化建议

### 功能增强

1. **批量同步**
   - 支持批量同步多个域名
   - 减少网络开销

2. **同步状态查询**
   - 提供查询同步状态的接口
   - 便于排查问题

3. **同步历史记录**
   - 保存同步操作的历史记录
   - 支持审计和回溯

4. **手动触发同步**
   - 提供管理界面手动触发同步
   - 便于修复遗漏的域名

### 技术优化

1. **自动重试机制**
   - 失败后自动重试
   - 使用指数退避策略

2. **监控面板**
   - 在管理界面展示同步统计信息
   - 可视化成功率和延迟

3. **配置热更新**
   - 支持不重启服务更新配置
   - 提高运维效率

4. **API 降级**
   - 当外部 API 长时间不可用时降级
   - 避免持续失败

## 9. 附录

### API 调用示例

```bash
# 请求格式
curl 'https://xxx' \
  -X 'PUT' \
  --data-raw 'DOMAIN-SUFFIX,example.com,Proxy'

# Go 代码示例
client := &http.Client{
    Timeout: 5 * time.Second,
}

req, _ := http.NewRequest("PUT", apiURL, strings.NewReader("DOMAIN-SUFFIX,example.com,Proxy"))
resp, err := client.Do(req)
if err != nil {
    // 记录错误日志
    log.Error("Failed to sync domain", "error", err)
    return
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    // 记录非 200 响应
    log.Error("API returned non-200 status", "status", resp.StatusCode)
}
```

### 域名提取示例

```go
import "golang.org/x/net/publicsuffix"

func ExtractRootDomain(domain string) (string, error) {
    // 移除协议前缀
    domain = strings.TrimPrefix(domain, "https://")
    domain = strings.TrimPrefix(domain, "http://")
    // 移除路径
    if idx := strings.Index(domain, "/"); idx != -1 {
        domain = domain[:idx]
    }
    // 提取根域名
    return publicsuffix.EffectiveTLDPlusOne(domain)
}
```

### 配置文件示例

```yaml
# configs/config.yaml
env_sync:
  api_url: "https://your-api-endpoint.com"
  timeout: 5s
  enabled: true
```
