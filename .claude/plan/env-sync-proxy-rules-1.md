# 环境变量同步功能增强规划 v1.1

## 已明确的决策

基于用户确认的技术选择：

1. **环境变量命名策略**：采用方案A
   - `OCTOPUS_ENV_SYNC_PROXY_RULE` 和 `OCTOPUS_ENV_SYNC_DIRECT_RULE`
   - 清晰的语义化命名，便于理解和维护

2. **默认值策略**：用户特殊要求
   - 两个环境变量都默认为 "DIRECT"
   - 不采用常规的PROXY/DIRECT分别默认策略

3. **API方法签名**：采用方案A
   - 修改现有方法 `SyncDomain(domain string, useProxy bool)`
   - 保持API简洁性和向后兼容性

4. **功能范围**：采用方案B
   - 同时实现两种策略配置（基于域名规则和全局开关）
   - 提供完整的代理控制能力

## 整体规划概述

### 项目目标

为Octopus项目的环境变量同步功能添加灵活的代理控制机制，允许用户通过环境变量配置特定域名的代理使用策略，同时保持现有API的简洁性和兼容性。

### 技术栈

- Go 1.21+
- 正则表达式匹配
- 环境变量配置
- HTTP代理控制

### 主要阶段

1. **核心基础设施实现**：代理规则管理器和配置解析
2. **API集成改造**：修改现有SyncDomain方法集成新的代理控制逻辑
3. **测试验证**：单元测试和集成测试确保功能正确性

## 详细技术设计

### 环境变量定义

```go
const (
    // 代理规则环境变量
    EnvProxyRule  = "OCTOPUS_ENV_SYNC_PROXY_RULE"
    EnvDirectRule = "OCTOPUS_ENV_SYNC_DIRECT_RULE"

    // 默认策略：用户要求都默认为DIRECT
    DefaultProxyRule  = "DIRECT"
    DefaultDirectRule = "DIRECT"
)
```

### 规则配置格式

支持多种规则格式，用逗号分隔：
- 通配符：`*.github.com,*.gitlab.com`
- 正则表达式：`^.*\.(github|gitlab)\.com$`
- 精确匹配：`api.github.com,raw.githubusercontent.com`

### 核心组件设计

#### ProxyRuleManager 结构

```go
type ProxyRuleManager struct {
    proxyRules  []Rule
    directRules []Rule
    mutex       sync.RWMutex
}

type Rule struct {
    Pattern   string
    IsRegex   bool
    CompiledRegex *regexp.Regexp // 仅当IsRegex为true时使用
}
```

## 详细任务分解

### 阶段 1：核心基础设施实现

#### 任务 1.1：创建代理规则管理器

- **目标**：实现ProxyRuleManager核心结构和方法
- **输入**：环境变量配置字符串
- **输出**：完整的规则管理器实现
- **涉及文件**：
  - `internal/env/proxy_rule_manager.go` (新建)
  - `internal/env/constants.go` (可能需要新建或修改)
- **预估工作量**：2-3小时

**实现要点**：
- 规则解析和编译逻辑
- 线程安全的规则管理
- 支持通配符和正则表达式
- 规则优先级处理

#### 任务 1.2：实现域名匹配算法

- **目标**：实现高效的域名规则匹配逻辑
- **输入**：域名字符串和规则集合
- **输出**：匹配结果（PROXY/DIRECT）
- **涉及文件**：
  - `internal/env/proxy_rule_manager.go`
- **预估工作量**：1-2小时

**实现要点**：
- 通配符匹配算法
- 正则表达式匹配
- 性能优化（缓存机制）

#### 任务 1.3：环境变量配置加载

- **目标**：实现环境变量读取和初始化逻辑
- **输入**：系统环境变量
- **输出**：初始化的ProxyRuleManager实例
- **涉及文件**：
  - `internal/env/config.go` (可能需要新建或修改)
  - `internal/env/proxy_rule_manager.go`
- **预估工作量**：1小时

**实现要点**：
- 环境变量读取和解析
- 默认值处理（都为DIRECT）
- 错误处理和日志记录

### 阶段 2：API集成改造

#### 任务 2.1：修改SyncDomain方法签名和实现

- **目标**：集成代理规则管理器到现有API
- **输入**：域名字符串和可选的代理偏好
- **输出**：基于规则的代理决策结果
- **涉及文件**：
  - 查找并修改包含SyncDomain方法的文件
  - 可能涉及接口定义文件
- **预估工作量**：2-3小时

**实现逻辑**：
```go
func SyncDomain(domain string, useProxy bool) ProxyDecision {
    ruleManager := GetProxyRuleManager()

    // 1. 首先检查精确的用户偏好
    if useProxy {
        // 检查是否在DIRECT规则中
        if ruleManager.ShouldUseDirect(domain) {
            return ProxyDecision{UseProxy: false, Reason: "DirectRule"}
        }
        return ProxyDecision{UseProxy: true, Reason: "UserPreference"}
    }

    // 2. 应用环境变量规则
    if ruleManager.ShouldUseProxy(domain) {
        return ProxyDecision{UseProxy: true, Reason: "ProxyRule"}
    }

    if ruleManager.ShouldUseDirect(domain) {
        return ProxyDecision{UseProxy: false, Reason: "DirectRule"}
    }

    // 3. 默认策略（DIRECT）
    return ProxyDecision{UseProxy: false, Reason: "Default"}
}
```

#### 任务 2.2：更新调用方代码

- **目标**：更新所有SyncDomain的调用点以适配新的决策机制
- **输入**：现有的SyncDomain调用代码
- **输出**：更新后的调用代码
- **涉及文件**：
  - 搜索所有调用SyncDomain的文件
- **预估工作量**：1-2小时

#### 任务 2.3：添加配置验证和错误处理

- **目标**：确保环境变量配置的健壮性
- **输入**：环境变量配置
- **输出**：验证结果和错误信息
- **涉及文件**：
  - `internal/env/proxy_rule_manager.go`
  - 可能需要添加配置验证工具
- **预估工作量**：1小时

### 阶段 3：测试验证

#### 任务 3.1：单元测试

- **目标**：为核心组件编写完整的单元测试
- **输入**：实现的代码模块
- **输出**：完整的测试套件
- **涉及文件**：
  - `internal/env/proxy_rule_manager_test.go` (新建)
  - `internal/env/config_test.go` (新建或修改)
- **预估工作量**：3-4小时

**测试覆盖点**：
- 规则解析和匹配
- 环境变量加载
- 边界条件处理
- 并发安全性

#### 任务 3.2：集成测试

- **目标**：验证整体功能的正确性
- **输入**：完整的功能实现
- **输出**：集成测试用例
- **涉及文件**：
  - `test/integration/env_sync_test.go` (可能新建)
- **预估工作量**：2-3小时

**测试场景**：
- 不同环境变量配置下的行为
- API调用的正确性
- 性能测试

#### 任务 3.3：文档和示例

- **目标**：提供使用文档和配置示例
- **输入**：实现的功能
- **输出**：README更新和示例配置
- **涉及文件**：
  - `README.md` (更新)
  - `docs/env-sync-proxy.md` (可能新建)
- **预估工作量**：1小时

## 实现细节

### 配置示例

```bash
# 示例1：基本配置
export OCTOPUS_ENV_SYNC_PROXY_RULE="*.github.com,*.gitlab.com"
export OCTOPUS_ENV_SYNC_DIRECT_RULE="localhost,*.local"

# 示例2：正则表达式配置
export OCTOPUS_ENV_SYNC_PROXY_RULE="^.*\.(github|gitlab)\.com$"
export OCTOPUS_ENV_SYNC_DIRECT_RULE="^(localhost|127\.0\.0\.1)$"

# 示例3：用户默认要求（都为DIRECT）
export OCTOPUS_ENV_SYNC_PROXY_RULE="DIRECT"
export OCTOPUS_ENV_SYNC_DIRECT_RULE="DIRECT"
```

### 性能考虑

1. **规则编译缓存**：正则表达式预编译
2. **匹配结果缓存**：热点域名结果缓存
3. **读写锁优化**：读多写少场景优化

### 错误处理策略

1. **配置解析错误**：记录警告，使用默认值
2. **正则表达式错误**：跳过错误规则，继续处理
3. **运行时错误**：降级到默认策略

## 风险评估

### 技术风险

1. **性能影响**：✅ 低风险
   - 规则匹配算法经过优化
   - 缓存机制减少重复计算

2. **兼容性问题**：✅ 低风险
   - 保持现有API签名
   - 向后兼容的默认行为

3. **配置复杂性**：⚠️ 中等风险
   - 提供清晰的文档和示例
   - 实现配置验证机制

### 缓解措施

1. **性能监控**：添加性能指标收集
2. **渐进式部署**：先在测试环境验证
3. **回退机制**：保留原有的简单代理控制逻辑

## 成功标准

1. ✅ 环境变量正确控制代理行为
2. ✅ 现有API保持兼容性
3. ✅ 配置灵活且易于理解
4. ✅ 测试覆盖率达到90%以上
5. ✅ 性能影响小于5%
6. ✅ 文档完整且清晰

## 交付物清单

1. **代码实现**
   - ProxyRuleManager核心组件
   - 修改后的SyncDomain方法
   - 配置管理模块

2. **测试套件**
   - 单元测试（覆盖率>90%）
   - 集成测试
   - 性能测试

3. **文档**
   - API文档更新
   - 配置指南
   - 迁移指南

4. **示例配置**
   - 常见场景配置模板
   - 最佳实践指南

---

**规划完成，准备进入实施阶段** ✅

此规划文档体现了用户的所有技术决策，特别是"都默认DIRECT"的特殊要求已在环境变量定义部分明确体现。所有不确定性已消除，可以直接进入代码实现阶段。