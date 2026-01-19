# 敏感信息过滤规则导出/导入修复规划

## 1. 问题分析

### 1.1 当前问题
- 数据导出时，敏感信息过滤规则（`SensitiveFilterRule`）未被包含在备份文件中
- 数据导入时，即使备份文件包含敏感信息过滤规则，也不会被恢复到系统

### 1.2 影响范围
- 影响所有使用数据导出/导入功能的用户
- 可能导致用户自定义的敏感信息过滤规则丢失
- 影响系统的敏感信息保护能力

### 1.3 根本原因
- `DBDump` 结构体未包含 `SensitiveFilterRules` 字段
- `DBExportAll` 函数未查询和导出 `SensitiveFilterRule` 表
- `DBImportIncremental` 函数未导入 `SensitiveFilterRule` 数据

## 2. 修复方案

### 2.1 技术决策
| 决策项 | 选择 | 说明 |
|--------|------|------|
| 内置规则处理 | 方案 B | 导入时更新内置规则的 Enabled 等字段，保留用户配置 |
| 版本兼容性 | 方案 A | 直接添加新字段，不考虑向后兼容 |

### 2.2 修改文件清单
| 文件 | 修改内容 |
|------|----------|
| `internal/model/backup.go` | 在 `DBDump` 结构中添加 `SensitiveFilterRules` 字段 |
| `internal/op/backup.go` | 修改导出和导入函数，处理敏感信息过滤规则 |

### 2.3 导入策略详细说明

**内置规则处理逻辑：**
- 对于 `BuiltIn=true` 的规则，使用 `OnConflict` 更新 `Enabled` 和 `Priority` 字段
- 对于 `BuiltIn=false` 的规则，使用 `OnConflict` 的 `DoNothing` 策略
- 避免覆盖内置规则的 `Name`、`Pattern`、`Replacement` 等核心字段

**实现方式：**
```go
// 内置规则：仅更新 Enabled 和 Priority
db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "id"}},
    DoUpdates: clause.AssignmentColumns([]string{"enabled", "priority"}),
}).Create(&builtInRules)

// 自定义规则：跳过已存在的
db.Clauses(clause.OnConflict{
    DoNothing: true,
}).Create(&customRules)
```

## 3. 实施步骤

### 步骤 1：修改 DBDump 结构（优先级：高）
- 在 `internal/model/backup.go` 中添加 `SensitiveFilterRules` 字段
- 字段位置放在 `Settings` 之后

### 步骤 2：修改导出函数（优先级：高）
- 在 `DBExportAll` 函数中添加查询 `SensitiveFilterRule` 的逻辑
- 导出所有规则（包括内置和自定义）

### 步骤 3：修改导入函数（优先级：高）
- 在 `DBImportIncremental` 函数中添加导入逻辑
- 区分内置规则和自定义规则，分别处理

### 步骤 4：测试验证（优先级：中）
- 编写测试用例
- 验证导出/导入功能
- 测试边界情况

## 4. 代码修改点

### 4.1 internal/model/backup.go

```go
type DBDump struct {
    Version      int       `json:"version"`
    ExportedAt   time.Time `json:"exported_at"`
    IncludeLogs  bool      `json:"include_logs"`
    IncludeStats bool      `json:"include_stats"`

    Channels   []Channel   `json:"channels,omitempty"`
    Groups     []Group     `json:"groups,omitempty"`
    GroupItems []GroupItem `json:"group_items,omitempty"`
    LLMInfos   []LLMInfo   `json:"llm_infos,omitempty"`
    APIKeys    []APIKey    `json:"api_keys,omitempty"`
    Settings   []Setting   `json:"settings,omitempty"`
    SensitiveFilterRules []SensitiveFilterRule `json:"sensitive_filter_rules,omitempty"` // 新增

    StatsTotal   []StatsTotal   `json:"stats_total,omitempty"`
    StatsDaily   []StatsDaily   `json:"stats_daily,omitempty"`
    StatsHourly  []StatsHourly  `json:"stats_hourly,omitempty"`
    StatsModel   []StatsModel   `json:"stats_model,omitempty"`
    StatsChannel []StatsChannel `json:"stats_channel,omitempty"`
    StatsAPIKey  []StatsAPIKey  `json:"stats_api_key,omitempty"`

    RelayLogs []RelayLog `json:"relay_logs,omitempty"`
}
```

### 4.2 internal/op/backup.go - 导出函数

在 `DBExportAll` 函数中添加：
```go
if err := conn.Find(&d.SensitiveFilterRules).Error; err != nil {
    return nil, fmt.Errorf("export sensitive_filter_rules: %w", err)
}
```

### 4.3 internal/op/backup.go - 导入函数

在 `DBImportIncremental` 函数的 Transaction 中添加：
```go
// 分离内置规则和自定义规则
var builtInRules []model.SensitiveFilterRule
var customRules []model.SensitiveFilterRule
for _, rule := range dump.SensitiveFilterRules {
    if rule.BuiltIn {
        builtInRules = append(builtInRules, rule)
    } else {
        customRules = append(customRules, rule)
    }
}

// 导入自定义规则（仅新增，不覆盖）
if n, err := createDoNothing(tx, customRules); err != nil {
    return fmt.Errorf("import custom sensitive_filter_rules: %w", err)
} else {
    res.RowsAffected["sensitive_filter_rules_custom"] = n
}

// 导入内置规则（仅更新 enabled 和 priority）
if len(builtInRules) > 0 {
    if n, err := createUpsertBuiltInRules(tx, builtInRules); err != nil {
        return fmt.Errorf("import builtin sensitive_filter_rules: %w", err)
    } else {
        res.RowsAffected["sensitive_filter_rules_builtin"] = n
    }
}
```

新增辅助函数：
```go
func createUpsertBuiltInRules(tx *gorm.DB, rows []model.SensitiveFilterRule) (int64, error) {
    if len(rows) == 0 {
        return 0, nil
    }
    result := tx.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "id"}},
        DoUpdates: clause.AssignmentColumns([]string{"enabled", "priority"}),
    }).Create(&rows)
    return result.RowsAffected, result.Error
}
```

## 5. 测试计划

### 5.1 单元测试

**测试用例 1：导出包含敏感信息过滤规则**
- 创建多个规则（内置和自定义）
- 执行导出
- 验证导出文件包含所有规则

**测试用例 2：导入自定义规则**
- 导入包含自定义规则的备份
- 验证规则被正确创建
- 验证重复导入不会出错

**测试用例 3：导入内置规则**
- 修改内置规则的 Enabled 状态
- 导入备份
- 验证内置规则的 Enabled 和 Priority 被更新
- 验证 Name、Pattern、Replacement 未被修改

**测试用例 4：空规则列表**
- 导入不包含敏感信息过滤规则的旧版本备份
- 验证不会报错

### 5.2 集成测试

**测试场景 1：完整导出-导入流程**
1. 创建测试数据（渠道、分组、敏感词规则等）
2. 执行导出
3. 清空数据库
4. 执行导入
5. 验证数据完整性

**测试场景 2：增量导入**
1. 创建初始数据
2. 导入备份（包含新规则）
3. 验证新旧规则都存在

### 5.3 边界情况测试

- 空规则列表
- 无效的规则数据
- 重复的规则 ID
- 超长的规则模式

## 6. 验收标准

### 6.1 功能要求
- ✅ 导出的备份文件必须包含所有敏感信息过滤规则
- ✅ 导入后系统中的规则必须与备份中的规则一致
- ✅ 内置规则的核心字段（Name、Pattern、Replacement）不会被覆盖
- ✅ 内置规则的可配置字段（Enabled、Priority）可以被更新
- ✅ 自定义规则正常导入，重复导入不会报错

### 6.2 性能要求
- 导出时间增加不超过 5%
- 导入时间增加不超过 5%
- 内存使用增长合理

### 6.3 兼容性要求
- 新版本可以处理不包含敏感信息过滤规则的旧版本备份
- 导入旧版本备份时不会报错

## 7. 风险和注意事项

### 7.1 技术风险
- **内置规则 ID 变化**：如果未来内置规则 ID 发生变化，可能导致导入失败
  - 缓解：使用 Name 而非 ID 作为冲突判断键
- **规则冲突**：用户自定义规则可能与内置规则模式冲突
  - 缓解：文档中明确说明规则优先级

### 7.2 业务风险
- **配置丢失**：用户修改的内置规则可能未正确保存
  - 缓解：导出时包含所有规则，包括内置规则
- **误操作**：用户可能意外导入错误的备份
  - 缓解：导入前显示预览信息

## 8. 实施时间表

| 步骤 | 任务 | 预计时间 |
|------|------|----------|
| 1 | 修改 DBDump 结构 | 0.5 小时 |
| 2 | 修改导出函数 | 0.5 小时 |
| 3 | 修改导入函数 | 1 小时 |
| 4 | 编写测试 | 1 小时 |
| 5 | 验证功能 | 0.5 小时 |
| **总计** | | **3.5 小时** |

## 9. 后续优化建议

1. **导出过滤选项**：允许用户选择是否导出内置规则
2. **导入预览**：导入前显示将要导入的规则列表
3. **规则验证**：导入前验证规则的正则表达式语法
4. **批量管理**：提供批量启用/禁用规则的功能
