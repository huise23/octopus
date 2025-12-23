package op

import (
	"context"
	"regexp"
	"sync"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

var (
	sensitiveRulesCache     []*CompiledRule
	sensitiveRulesCacheLock sync.RWMutex
	sensitiveFilterEnabled  bool
)

// CompiledRule 编译后的规则
type CompiledRule struct {
	Rule  *model.SensitiveFilterRule
	Regex *regexp.Regexp
}

// SensitiveFilterInit 初始化敏感过滤规则
func SensitiveFilterInit() {
	// 检查是否需要初始化内置规则
	var count int64
	db.GetDB().Model(&model.SensitiveFilterRule{}).Count(&count)
	if count == 0 {
		rules := model.DefaultSensitiveFilterRules()
		for _, rule := range rules {
			if err := db.GetDB().Create(&rule).Error; err != nil {
				log.Warnf("failed to create default sensitive filter rule: %v", err)
			}
		}
		log.Infof("initialized %d default sensitive filter rules", len(rules))
	}

	// 加载规则到缓存
	SensitiveFilterRefresh()
}

// SensitiveFilterRefresh 刷新缓存
func SensitiveFilterRefresh() {
	// 获取全局开关状态
	setting, err := SettingGet(model.SettingKeySensitiveFilterEnabled)
	if err != nil {
		sensitiveFilterEnabled = true // 默认启用
	} else {
		sensitiveFilterEnabled = setting.Value == "true"
	}

	// 加载启用的规则
	var rules []model.SensitiveFilterRule
	if err := db.GetDB().Where("enabled = ?", true).Order("priority DESC").Find(&rules).Error; err != nil {
		log.Warnf("failed to load sensitive filter rules: %v", err)
		return
	}

	compiled := make([]*CompiledRule, 0, len(rules))
	for i := range rules {
		regex, err := regexp.Compile(rules[i].Pattern)
		if err != nil {
			log.Warnf("invalid regex pattern for rule %s: %v", rules[i].Name, err)
			continue
		}
		compiled = append(compiled, &CompiledRule{
			Rule:  &rules[i],
			Regex: regex,
		})
	}

	sensitiveRulesCacheLock.Lock()
	sensitiveRulesCache = compiled
	sensitiveRulesCacheLock.Unlock()

	log.Infof("loaded %d sensitive filter rules, global enabled: %v", len(compiled), sensitiveFilterEnabled)
}

// SensitiveFilterGetEnabled 获取是否启用过滤
func SensitiveFilterGetEnabled() bool {
	sensitiveRulesCacheLock.RLock()
	defer sensitiveRulesCacheLock.RUnlock()
	return sensitiveFilterEnabled
}

// SensitiveFilterText 过滤文本中的敏感信息
func SensitiveFilterText(text string) (string, int) {
	sensitiveRulesCacheLock.RLock()
	rules := sensitiveRulesCache
	enabled := sensitiveFilterEnabled
	sensitiveRulesCacheLock.RUnlock()

	if !enabled || len(rules) == 0 {
		return text, 0
	}

	count := 0
	result := text
	for _, rule := range rules {
		if rule.Regex.MatchString(result) {
			result = rule.Regex.ReplaceAllString(result, rule.Rule.Replacement)
			count++
		}
	}
	return result, count
}

// SensitiveFilterRuleList 获取所有规则
func SensitiveFilterRuleList(ctx context.Context) ([]model.SensitiveFilterRule, error) {
	var rules []model.SensitiveFilterRule
	if err := db.GetDB().WithContext(ctx).Order("priority DESC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// SensitiveFilterRuleCreate 创建规则
func SensitiveFilterRuleCreate(rule *model.SensitiveFilterRule, ctx context.Context) error {
	// 验证正则表达式
	if _, err := regexp.Compile(rule.Pattern); err != nil {
		return err
	}
	rule.BuiltIn = false
	if err := db.GetDB().WithContext(ctx).Create(rule).Error; err != nil {
		return err
	}
	SensitiveFilterRefresh()
	return nil
}

// SensitiveFilterRuleUpdate 更新规则
func SensitiveFilterRuleUpdate(rule *model.SensitiveFilterRule, ctx context.Context) error {
	// 验证正则表达式
	if _, err := regexp.Compile(rule.Pattern); err != nil {
		return err
	}

	// 内置规则只能修改 Enabled
	var existing model.SensitiveFilterRule
	if err := db.GetDB().WithContext(ctx).First(&existing, rule.ID).Error; err != nil {
		return err
	}

	if existing.BuiltIn {
		if err := db.GetDB().WithContext(ctx).Model(rule).Update("enabled", rule.Enabled).Error; err != nil {
			return err
		}
	} else {
		if err := db.GetDB().WithContext(ctx).Save(rule).Error; err != nil {
			return err
		}
	}
	SensitiveFilterRefresh()
	return nil
}

// SensitiveFilterRuleDelete 删除规则
func SensitiveFilterRuleDelete(id int, ctx context.Context) error {
	// 内置规则不能删除
	var rule model.SensitiveFilterRule
	if err := db.GetDB().WithContext(ctx).First(&rule, id).Error; err != nil {
		return err
	}
	if rule.BuiltIn {
		return nil // 静默忽略
	}

	if err := db.GetDB().WithContext(ctx).Delete(&model.SensitiveFilterRule{}, id).Error; err != nil {
		return err
	}
	SensitiveFilterRefresh()
	return nil
}

// SensitiveFilterRuleToggle 切换规则启用状态
func SensitiveFilterRuleToggle(id int, enabled bool, ctx context.Context) error {
	if err := db.GetDB().WithContext(ctx).Model(&model.SensitiveFilterRule{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return err
	}
	SensitiveFilterRefresh()
	return nil
}
