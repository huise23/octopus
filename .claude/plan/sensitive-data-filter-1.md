# 请求转发敏感信息过滤功能规划 v2

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

## 设计变更（v2）

### 变更点

1. **规则持久化** - 过滤规则存储到数据库，支持增删改查
2. **内存缓存** - 规则加载到内存，前端操作时自动刷新缓存
3. **独立启用** - 每条规则可单独启用/禁用，方便调试
4. **设置页集成** - 规则管理 UI 放在设置页面

## 数据模型

### 敏感过滤规则表 (`sensitive_filter_rule`)

```go
type SensitiveFilterRule struct {
    ID          int    `json:"id" gorm:"primaryKey"`
    Name        string `json:"name" gorm:"not null"`           // 规则名称
    Pattern     string `json:"pattern" gorm:"not null"`        // 正则表达式
    Replacement string `json:"replacement" gorm:"not null"`    // 替换文本
    Enabled     bool   `json:"enabled" gorm:"default:true"`    // 是否启用
    BuiltIn     bool   `json:"built_in" gorm:"default:false"`  // 是否内置规则
    Priority    int    `json:"priority" gorm:"default:0"`      // 优先级（越大越先匹配）
}
```

### 内置规则（首次启动自动创建）

| Name | Pattern | Replacement | Enabled |
|------|---------|-------------|---------|
| OpenAI API Key | `sk-[a-zA-Z0-9_-]{20,}` | `[FILTERED:API_KEY]` | true |
| Anthropic API Key | `sk-ant-[a-zA-Z0-9_-]{20,}` | `[FILTERED:API_KEY]` | true |
| Database URL | `(mysql\|postgres\|mongodb\|redis)://[^\s"'<>]+` | `[FILTERED:DB_URL]` | true |
| Bearer Token | `Bearer\s+[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+` | `[FILTERED:TOKEN]` | true |
| AWS Access Key | `AKIA[0-9A-Z]{16}` | `[FILTERED:AWS_KEY]` | true |
| GitHub Token | `ghp_[a-zA-Z0-9]{36}` | `[FILTERED:GH_TOKEN]` | true |
| Private Key | `-----BEGIN[A-Z ]*PRIVATE KEY-----` | `[FILTERED:PRIVATE_KEY]` | true |
| Password JSON | `"password"\s*:\s*"[^"]*"` | `"password":"[FILTERED]"` | false |

## 实施方案

### 文件结构

```
internal/
├── model/
│   └── sensitive.go           # 规则数据模型
├── op/
│   └── sensitive.go           # 规则 CRUD + 缓存
├── server/handlers/
│   └── sensitive.go           # API 接口
├── relay/
│   └── relay.go               # 调用过滤

web/src/
├── api/endpoints/
│   └── sensitive.ts           # API 调用
├── components/modules/setting/
│   └── Sensitive.tsx          # 规则管理 UI
```

### 后端实现

#### 1. 数据模型 (`internal/model/sensitive.go`)

```go
package model

type SensitiveFilterRule struct {
    ID          int    `json:"id" gorm:"primaryKey"`
    Name        string `json:"name" gorm:"not null"`
    Pattern     string `json:"pattern" gorm:"not null"`
    Replacement string `json:"replacement" gorm:"not null"`
    Enabled     bool   `json:"enabled" gorm:"default:true"`
    BuiltIn     bool   `json:"built_in" gorm:"default:false"`
    Priority    int    `json:"priority" gorm:"default:0"`
}

func DefaultSensitiveFilterRules() []SensitiveFilterRule
```

#### 2. 缓存与操作 (`internal/op/sensitive.go`)

```go
package op

var (
    sensitiveRulesCache     []*CompiledRule
    sensitiveRulesCacheLock sync.RWMutex
)

type CompiledRule struct {
    Rule    *model.SensitiveFilterRule
    Regex   *regexp.Regexp
}

// 初始化：加载规则到缓存
func SensitiveFilterInit()

// 刷新缓存（增删改后调用）
func SensitiveFilterRefresh()

// 获取启用的规则（从缓存）
func SensitiveFilterGetEnabled() []*CompiledRule

// CRUD
func SensitiveFilterRuleList() ([]model.SensitiveFilterRule, error)
func SensitiveFilterRuleCreate(rule *model.SensitiveFilterRule) error
func SensitiveFilterRuleUpdate(rule *model.SensitiveFilterRule) error
func SensitiveFilterRuleDelete(id int) error
func SensitiveFilterRuleToggle(id int, enabled bool) error

// 过滤文本
func SensitiveFilterText(text string) (string, int)
```

#### 3. API 接口 (`internal/server/handlers/sensitive.go`)

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/sensitive/list` | 获取规则列表 |
| POST | `/api/v1/sensitive/create` | 创建规则 |
| POST | `/api/v1/sensitive/update` | 更新规则 |
| DELETE | `/api/v1/sensitive/delete/:id` | 删除规则 |
| POST | `/api/v1/sensitive/toggle/:id` | 切换启用状态 |

#### 4. 集成到转发 (`internal/relay/relay.go`)

```go
func parseRequest(...) {
    // ... 解析请求

    // 敏感信息过滤
    filterMessages(internalRequest)

    return internalRequest, inAdapter, nil
}

func filterMessages(req *model.InternalLLMRequest) {
    for i := range req.Messages {
        if req.Messages[i].Content.Content != nil {
            filtered, _ := op.SensitiveFilterText(*req.Messages[i].Content.Content)
            req.Messages[i].Content.Content = &filtered
        }
        // 处理 MultipleContent...
    }
}
```

### 前端实现

#### 设置页 - 敏感过滤规则 (`web/src/components/modules/setting/Sensitive.tsx`)

**UI 布局：**
```
┌─────────────────────────────────────────────────────┐
│ 🔒 敏感信息过滤                          [+ 添加规则] │
├─────────────────────────────────────────────────────┤
│ ☑ OpenAI API Key          sk-[a-zA-Z0-9_-]{20,}    │
│ ☑ Database URL            (mysql|postgres)://...   │
│ ☐ Password JSON           "password":...    [🗑️]   │
│ ...                                                 │
└─────────────────────────────────────────────────────┘
```

**功能：**
- 列表展示所有规则（名称、模式、启用状态）
- 开关切换单条规则启用/禁用
- 添加自定义规则（弹窗表单）
- 编辑规则（内置规则仅可改启用状态）
- 删除规则（内置规则不可删除）

## 文件修改清单

| 文件 | 操作 | 说明 |
|-----|------|------|
| `internal/model/sensitive.go` | 新建 | 规则数据模型 |
| `internal/op/sensitive.go` | 新建 | 规则 CRUD + 缓存 |
| `internal/server/handlers/sensitive.go` | 新建 | API 接口 |
| `internal/server/router/router.go` | 修改 | 注册路由 |
| `internal/db/db.go` | 修改 | 自动迁移表 |
| `internal/relay/relay.go` | 修改 | 调用过滤 |
| `web/src/api/endpoints/sensitive.ts` | 新建 | API 调用 |
| `web/src/components/modules/setting/Sensitive.tsx` | 新建 | 规则管理 UI |
| `web/src/components/modules/setting/index.tsx` | 修改 | 引入组件 |
| `web/public/locale/zh.json` | 修改 | 中文翻译 |
| `web/public/locale/en.json` | 修改 | 英文翻译 |

## 验收标准

1. [ ] 规则存储到数据库，支持增删改查
2. [ ] 规则缓存到内存，操作后自动刷新
3. [ ] 每条规则可单独启用/禁用
4. [ ] 内置规则首次启动自动创建
5. [ ] 内置规则不可删除，仅可改启用状态
6. [ ] 请求转发前按启用规则过滤
7. [ ] 设置页面可管理规则列表

## 实施顺序

1. **后端模型 + CRUD + 缓存**
2. **后端 API 接口**
3. **集成到 relay 转发**
4. **前端规则管理 UI**
5. **国际化翻译**
