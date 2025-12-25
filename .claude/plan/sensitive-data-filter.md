# 请求转发敏感信息过滤功能规划

## 背景分析

### 问题场景

用户使用 Claude 等 AI 编码助手时，可能在对话中无意发送敏感信息：

```
用户: 帮我连接数据库，连接串是 mysql://root:MyP@ssw0rd123@db.example.com:3306/prod
用户: 这是我的 API Key: sk-proj-abc123xyz789，帮我调用一下
```

这些敏感信息会通过 Octopus 代理转发给 LLM 提供商（OpenAI、Anthropic 等），存在泄露风险。

### 数据流分析

```
用户请求 → Octopus (parseRequest) → InternalLLMRequest → forward → LLM 提供商
                                          ↑
                                    【过滤点】
```

**关键代码位置**：`internal/relay/relay.go`
- `parseRequest()` - 解析用户请求为 `InternalLLMRequest`
- `forward()` - 转发请求到上游 LLM

**需要过滤的字段**：
- `InternalLLMRequest.Messages[].Content` - 消息文本内容
- `InternalLLMRequest.Messages[].MultipleContent[].Text` - 多部分消息的文本

## 功能目标

1. **转发前过滤** - 在请求转发给 LLM 之前，检测并替换敏感信息
2. **可配置开关** - 允许用户启用/禁用过滤功能
3. **可配置规则** - 支持自定义敏感信息匹配模式
4. **检测提示** - 检测到敏感信息时记录日志或提示用户

## 敏感信息模式

### 内置模式（默认启用）

| 类型 | 正则模式 | 示例 |
|-----|---------|------|
| API Key (OpenAI) | `sk-[a-zA-Z0-9]{20,}` | `sk-proj-abc123...` |
| API Key (Anthropic) | `sk-ant-[a-zA-Z0-9-]{20,}` | `sk-ant-api03-...` |
| Bearer Token | `Bearer\s+[a-zA-Z0-9-_.]+` | `Bearer eyJhbG...` |
| 数据库连接串 | `(mysql\|postgres\|mongodb\|redis)://[^\s"']+` | `mysql://user:pass@host/db` |
| 密码字段 (JSON) | `"password"\s*:\s*"[^"]*"` | `"password": "secret123"` |
| 私钥 | `-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----` | PEM 格式私钥 |
| AWS Key | `AKIA[0-9A-Z]{16}` | AWS Access Key ID |
| GitHub Token | `ghp_[a-zA-Z0-9]{36}` | GitHub Personal Token |

### 替换策略

- 检测到敏感信息后替换为 `[FILTERED: <类型>]`
- 例如：`mysql://root:pass@host/db` → `[FILTERED: DATABASE_URL]`

## 实施方案

### 文件结构

```
internal/
├── utils/
│   └── sensitive/
│       ├── filter.go      # 过滤器核心逻辑
│       ├── patterns.go    # 内置敏感模式定义
│       └── filter_test.go # 单元测试
├── model/
│   └── setting.go         # 新增设置键
├── relay/
│   └── relay.go           # 调用过滤器
```

### 核心实现

#### 1. 过滤器 (`internal/utils/sensitive/filter.go`)

```go
package sensitive

type Filter struct {
    enabled  bool
    patterns []*Pattern
}

type Pattern struct {
    Name    string
    Regex   *regexp.Regexp
    Replace string
}

func NewFilter(enabled bool, customPatterns []string) *Filter

func (f *Filter) FilterRequest(req *model.InternalLLMRequest) (filtered bool, count int)

func (f *Filter) FilterText(text string) (string, bool)
```

#### 2. 设置键 (`internal/model/setting.go`)

```go
const (
    SettingKeySensitiveFilterEnabled  SettingKey = "sensitive_filter_enabled"   // 是否启用
    SettingKeySensitiveFilterPatterns SettingKey = "sensitive_filter_patterns"  // 自定义模式 JSON
)
```

#### 3. 调用位置 (`internal/relay/relay.go`)

```go
func parseRequest(...) {
    // ... 解析请求

    // 敏感信息过滤
    if filter := sensitive.GetFilter(); filter.Enabled() {
        if filtered, count := filter.FilterRequest(internalRequest); filtered {
            log.Warnf("Filtered %d sensitive patterns from request", count)
        }
    }

    return internalRequest, inAdapter, nil
}
```

### 前端配置界面

**位置**：设置页面新增"安全设置"卡片

**功能**：
- 开关：启用/禁用敏感信息过滤
- 列表：显示内置模式（只读）
- 输入：添加自定义模式

## 文件修改清单

| 文件 | 操作 | 说明 |
|-----|------|------|
| `internal/utils/sensitive/filter.go` | 新建 | 过滤器核心 |
| `internal/utils/sensitive/patterns.go` | 新建 | 内置模式 |
| `internal/model/setting.go` | 修改 | 新增设置键 |
| `internal/op/setting.go` | 修改 | 过滤器初始化 |
| `internal/relay/relay.go` | 修改 | 调用过滤器 |
| `web/src/components/modules/setting/Security.tsx` | 新建 | 安全设置 UI |
| `web/src/components/modules/setting/index.tsx` | 修改 | 引入安全设置 |
| `web/src/api/endpoints/setting.ts` | 修改 | 新增设置键 |
| `web/public/locale/zh.json` | 修改 | 中文翻译 |
| `web/public/locale/en.json` | 修改 | 英文翻译 |

## 验收标准

1. [ ] 请求中的敏感信息在转发前被过滤
2. [ ] 过滤后的请求正常转发，不影响 LLM 响应
3. [ ] 设置页面可开关过滤功能
4. [ ] 支持查看内置模式列表
5. [ ] 支持添加自定义过滤模式
6. [ ] 过滤时记录日志

## 优先级

1. **P0**: 后端过滤器核心实现 + 内置模式
2. **P0**: 集成到 relay 转发流程
3. **P1**: 设置页面开关
4. **P2**: 自定义模式配置 UI

## 风险与注意事项

| 风险 | 影响 | 缓解措施 |
|-----|-----|---------|
| 误过滤正常内容 | 中 | 模式设计保守，提供开关 |
| 性能影响 | 低 | 正则预编译，仅文本内容过滤 |
| 遗漏敏感信息 | 中 | 持续更新模式库，支持自定义 |
