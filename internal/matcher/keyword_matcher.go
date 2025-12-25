package matcher

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// KeywordMatcher 关键字匹配引擎
type KeywordMatcher struct {
	compiledRegexes map[string]*regexp.Regexp
	regexMutex      sync.RWMutex
}

// NewKeywordMatcher 创建新的关键字匹配器
func NewKeywordMatcher() *KeywordMatcher {
	return &KeywordMatcher{
		compiledRegexes: make(map[string]*regexp.Regexp),
	}
}

// MatchModel 根据分组配置匹配模型名称
func (km *KeywordMatcher) MatchModel(modelName string, group model.Group) bool {
	// 根据匹配模式进行匹配
	switch group.MatchMode {
	case model.GroupMatchModeNameOnly:
		return km.matchByName(modelName, group.Name)
	case model.GroupMatchModeKeywordOnly:
		return km.matchByKeywords(modelName, group.Keywords)
	case model.GroupMatchModeBoth:
		return km.matchByName(modelName, group.Name) || km.matchByKeywords(modelName, group.Keywords)
	default:
		// 默认兼容模式，使用分组名称匹配
		return km.matchByName(modelName, group.Name)
	}
}

// matchByName 通过分组名称进行模糊匹配（保持现有逻辑）
func (km *KeywordMatcher) matchByName(modelName, groupName string) bool {
	if groupName == "" {
		return false
	}
	return strings.Contains(strings.ToLower(modelName), strings.ToLower(groupName))
}

// matchByKeywords 通过关键字列表进行匹配
func (km *KeywordMatcher) matchByKeywords(modelName, keywordsJSON string) bool {
	if keywordsJSON == "" {
		return false
	}

	var keywords []model.GroupKeyword
	if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
		log.Errorf("failed to parse keywords JSON: %v", err)
		return false
	}

	// 任意一个关键字匹配成功即返回 true
	for _, keyword := range keywords {
		if km.matchSingleKeyword(modelName, keyword) {
			return true
		}
	}
	return false
}

// matchSingleKeyword 匹配单个关键字
func (km *KeywordMatcher) matchSingleKeyword(modelName string, keyword model.GroupKeyword) bool {
	if keyword.Pattern == "" {
		return false
	}

	switch keyword.Type {
	case "exact":
		return km.matchExact(modelName, keyword.Pattern)
	case "fuzzy":
		return km.matchFuzzy(modelName, keyword.Pattern)
	case "regex":
		return km.matchRegex(modelName, keyword.Pattern)
	default:
		// 默认使用模糊匹配
		return km.matchFuzzy(modelName, keyword.Pattern)
	}
}

// matchExact 精确匹配
func (km *KeywordMatcher) matchExact(modelName, pattern string) bool {
	return strings.EqualFold(modelName, pattern)
}

// matchFuzzy 模糊匹配
func (km *KeywordMatcher) matchFuzzy(modelName, pattern string) bool {
	return strings.Contains(strings.ToLower(modelName), strings.ToLower(pattern))
}

// matchRegex 正则表达式匹配
func (km *KeywordMatcher) matchRegex(modelName, pattern string) bool {
	regex := km.getCompiledRegex(pattern)
	if regex == nil {
		return false
	}
	return regex.MatchString(modelName)
}

// getCompiledRegex 获取编译后的正则表达式（带缓存）
func (km *KeywordMatcher) getCompiledRegex(pattern string) *regexp.Regexp {
	// 先尝试读锁获取
	km.regexMutex.RLock()
	regex, exists := km.compiledRegexes[pattern]
	km.regexMutex.RUnlock()

	if exists {
		return regex
	}

	// 需要编译新的正则表达式，使用写锁
	km.regexMutex.Lock()
	defer km.regexMutex.Unlock()

	// 双重检查，防止并发编译
	if regex, exists := km.compiledRegexes[pattern]; exists {
		return regex
	}

	// 编译正则表达式
	compiledRegex, err := regexp.Compile(pattern)
	if err != nil {
		log.Errorf("failed to compile regex pattern '%s': %v", pattern, err)
		// 缓存编译失败的结果，避免重复尝试
		km.compiledRegexes[pattern] = nil
		return nil
	}

	km.compiledRegexes[pattern] = compiledRegex
	return compiledRegex
}

// ValidateKeywords 验证关键字配置的有效性
func (km *KeywordMatcher) ValidateKeywords(keywordsJSON string) error {
	if keywordsJSON == "" {
		return nil
	}

	var keywords []model.GroupKeyword
	if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
		return err
	}

	// 验证每个关键字
	for i, keyword := range keywords {
		if keyword.Pattern == "" {
			continue
		}

		// 验证匹配类型
		switch keyword.Type {
		case "exact", "fuzzy", "":
			// 这些类型不需要特殊验证
		case "regex":
			// 验证正则表达式语法
			if _, err := regexp.Compile(keyword.Pattern); err != nil {
				return err
			}
		default:
			log.Warnf("unknown keyword type '%s' at index %d, will use fuzzy matching", keyword.Type, i)
		}
	}

	return nil
}

// TestMatch 测试匹配功能（用于调试和API测试）
func (km *KeywordMatcher) TestMatch(modelName string, keywords []model.GroupKeyword) []bool {
	results := make([]bool, len(keywords))
	for i, keyword := range keywords {
		results[i] = km.matchSingleKeyword(modelName, keyword)
	}
	return results
}

// ClearRegexCache 清理正则表达式缓存
func (km *KeywordMatcher) ClearRegexCache() {
	km.regexMutex.Lock()
	defer km.regexMutex.Unlock()
	km.compiledRegexes = make(map[string]*regexp.Regexp)
}

// GetCacheStats 获取缓存统计信息
func (km *KeywordMatcher) GetCacheStats() map[string]interface{} {
	km.regexMutex.RLock()
	defer km.regexMutex.RUnlock()

	compiled := 0
	failed := 0
	for _, regex := range km.compiledRegexes {
		if regex != nil {
			compiled++
		} else {
			failed++
		}
	}

	return map[string]interface{}{
		"total_patterns":    len(km.compiledRegexes),
		"compiled_success":  compiled,
		"compilation_failed": failed,
	}
}