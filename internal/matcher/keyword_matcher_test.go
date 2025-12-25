package matcher

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestKeywordMatcher_MatchModel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		group     model.Group
		expected  bool
	}{
		{
			name:      "仅分组名称匹配 - 成功",
			modelName: "gpt-4-turbo",
			group: model.Group{
				Name:      "gpt",
				MatchMode: model.GroupMatchModeNameOnly,
			},
			expected: true,
		},
		{
			name:      "仅分组名称匹配 - 失败",
			modelName: "claude-3-opus",
			group: model.Group{
				Name:      "gpt",
				MatchMode: model.GroupMatchModeNameOnly,
			},
			expected: false,
		},
		{
			name:      "精确匹配成功",
			modelName: "gpt-4",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"exact"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "精确匹配失败",
			modelName: "gpt-4-turbo",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"exact"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: false,
		},
		{
			name:      "模糊匹配成功",
			modelName: "gpt-4-turbo-preview",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "正则匹配成功 - Claude系列",
			modelName: "claude-3-opus-20240229",
			group: model.Group{
				Keywords:  `[{"pattern":"claude-3-.*","type":"regex"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "正则匹配成功 - GPT系列",
			modelName: "gpt-4-turbo-2024-04-09",
			group: model.Group{
				Keywords:  `[{"pattern":"^gpt-[0-9]+.*","type":"regex"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "正则匹配失败",
			modelName: "gemini-pro",
			group: model.Group{
				Keywords:  `[{"pattern":"^gpt-[0-9]+.*","type":"regex"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: false,
		},
		{
			name:      "多个关键字匹配 - 第一个匹配",
			modelName: "gpt-4-vision",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"fuzzy"},{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "多个关键字匹配 - 第二个匹配",
			modelName: "claude-3-sonnet",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"fuzzy"},{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "多个关键字匹配 - 都不匹配",
			modelName: "gemini-pro",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":"fuzzy"},{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: false,
		},
		{
			name:      "组合匹配模式 - 分组名称匹配",
			modelName: "gpt-3.5-turbo",
			group: model.Group{
				Name:      "gpt",
				Keywords:  `[{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeBoth,
			},
			expected: true,
		},
		{
			name:      "组合匹配模式 - 关键字匹配",
			modelName: "claude-3-haiku",
			group: model.Group{
				Name:      "gpt",
				Keywords:  `[{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeBoth,
			},
			expected: true,
		},
		{
			name:      "组合匹配模式 - 都不匹配",
			modelName: "gemini-pro",
			group: model.Group{
				Name:      "gpt",
				Keywords:  `[{"pattern":"claude","type":"fuzzy"}]`,
				MatchMode: model.GroupMatchModeBoth,
			},
			expected: false,
		},
		{
			name:      "默认匹配类型 - 使用模糊匹配",
			modelName: "gpt-4-turbo",
			group: model.Group{
				Keywords:  `[{"pattern":"gpt-4","type":""}]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: true,
		},
		{
			name:      "空关键字列表",
			modelName: "gpt-4",
			group: model.Group{
				Keywords:  `[]`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: false,
		},
		{
			name:      "无效JSON格式",
			modelName: "gpt-4",
			group: model.Group{
				Keywords:  `invalid json`,
				MatchMode: model.GroupMatchModeKeywordOnly,
			},
			expected: false,
		},
	}

	matcher := NewKeywordMatcher()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.MatchModel(tt.modelName, tt.group)
			assert.Equal(t, tt.expected, result, "Expected %v, got %v for model '%s'", tt.expected, result, tt.modelName)
		})
	}
}

func TestKeywordMatcher_matchExact(t *testing.T) {
	matcher := NewKeywordMatcher()

	tests := []struct {
		modelName string
		pattern   string
		expected  bool
	}{
		{"gpt-4", "gpt-4", true},
		{"GPT-4", "gpt-4", true},
		{"gpt-4-turbo", "gpt-4", false},
		{"claude-3", "claude-3", true},
		{"", "gpt-4", false},
		{"gpt-4", "", false},
	}

	for _, tt := range tests {
		result := matcher.matchExact(tt.modelName, tt.pattern)
		assert.Equal(t, tt.expected, result, "matchExact('%s', '%s') = %v, expected %v", tt.modelName, tt.pattern, result, tt.expected)
	}
}

func TestKeywordMatcher_matchFuzzy(t *testing.T) {
	matcher := NewKeywordMatcher()

	tests := []struct {
		modelName string
		pattern   string
		expected  bool
	}{
		{"gpt-4-turbo", "gpt", true},
		{"claude-3-opus", "claude", true},
		{"gemini-pro", "gpt", false},
		{"GPT-4-TURBO", "gpt", true},
		{"", "gpt", false},
		{"gpt-4", "", false},
	}

	for _, tt := range tests {
		result := matcher.matchFuzzy(tt.modelName, tt.pattern)
		assert.Equal(t, tt.expected, result, "matchFuzzy('%s', '%s') = %v, expected %v", tt.modelName, tt.pattern, result, tt.expected)
	}
}

func TestKeywordMatcher_matchRegex(t *testing.T) {
	matcher := NewKeywordMatcher()

	tests := []struct {
		modelName string
		pattern   string
		expected  bool
	}{
		{"gpt-4", "^gpt-[0-9]+$", true},
		{"gpt-4-turbo", "^gpt-[0-9]+$", false},
		{"gpt-4-turbo", "^gpt-[0-9]+.*", true},
		{"claude-3-opus-20240229", "claude-3-.*", true},
		{"gemini-pro", "claude-3-.*", false},
		{"invalid-model", "[invalid-regex", false}, // 无效正则表达式
	}

	for _, tt := range tests {
		result := matcher.matchRegex(tt.modelName, tt.pattern)
		assert.Equal(t, tt.expected, result, "matchRegex('%s', '%s') = %v, expected %v", tt.modelName, tt.pattern, result, tt.expected)
	}
}

func TestKeywordMatcher_ValidateKeywords(t *testing.T) {
	matcher := NewKeywordMatcher()

	tests := []struct {
		name        string
		keywordsJSON string
		expectError bool
	}{
		{
			name:        "有效的关键字配置",
			keywordsJSON: `[{"pattern":"gpt-4","type":"exact"},{"pattern":"claude","type":"fuzzy"}]`,
			expectError: false,
		},
		{
			name:        "有效的正则表达式",
			keywordsJSON: `[{"pattern":"^gpt-[0-9]+.*","type":"regex"}]`,
			expectError: false,
		},
		{
			name:        "无效的JSON格式",
			keywordsJSON: `invalid json`,
			expectError: true,
		},
		{
			name:        "无效的正则表达式",
			keywordsJSON: `[{"pattern":"[invalid-regex","type":"regex"}]`,
			expectError: true,
		},
		{
			name:        "空字符串",
			keywordsJSON: "",
			expectError: false,
		},
		{
			name:        "空数组",
			keywordsJSON: "[]",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matcher.ValidateKeywords(tt.keywordsJSON)
			if tt.expectError {
				assert.Error(t, err, "Expected error for keywords: %s", tt.keywordsJSON)
			} else {
				assert.NoError(t, err, "Expected no error for keywords: %s", tt.keywordsJSON)
			}
		})
	}
}

func TestKeywordMatcher_TestMatch(t *testing.T) {
	matcher := NewKeywordMatcher()

	keywords := []model.GroupKeyword{
		{Pattern: "gpt-4", Type: "exact"},
		{Pattern: "claude", Type: "fuzzy"},
		{Pattern: "^gemini-.*", Type: "regex"},
	}

	tests := []struct {
		modelName string
		expected  []bool
	}{
		{"gpt-4", []bool{true, false, false}},
		{"claude-3-opus", []bool{false, true, false}},
		{"gemini-pro", []bool{false, false, true}},
		{"unknown-model", []bool{false, false, false}},
	}

	for _, tt := range tests {
		result := matcher.TestMatch(tt.modelName, keywords)
		assert.Equal(t, tt.expected, result, "TestMatch('%s') = %v, expected %v", tt.modelName, result, tt.expected)
	}
}

func TestKeywordMatcher_RegexCache(t *testing.T) {
	matcher := NewKeywordMatcher()

	// 测试正则表达式缓存
	pattern := "^gpt-[0-9]+.*"
	modelName := "gpt-4-turbo"

	// 第一次匹配，应该编译并缓存正则表达式
	result1 := matcher.matchRegex(modelName, pattern)
	assert.True(t, result1)

	// 第二次匹配，应该使用缓存的正则表达式
	result2 := matcher.matchRegex(modelName, pattern)
	assert.True(t, result2)

	// 检查缓存统计
	stats := matcher.GetCacheStats()
	assert.Equal(t, 1, stats["total_patterns"])
	assert.Equal(t, 1, stats["compiled_success"])
	assert.Equal(t, 0, stats["compilation_failed"])

	// 测试无效正则表达式的缓存
	invalidPattern := "[invalid-regex"
	result3 := matcher.matchRegex(modelName, invalidPattern)
	assert.False(t, result3)

	// 再次使用相同的无效正则表达式
	result4 := matcher.matchRegex(modelName, invalidPattern)
	assert.False(t, result4)

	// 检查缓存统计
	stats = matcher.GetCacheStats()
	assert.Equal(t, 2, stats["total_patterns"])
	assert.Equal(t, 1, stats["compiled_success"])
	assert.Equal(t, 1, stats["compilation_failed"])

	// 清理缓存
	matcher.ClearRegexCache()
	stats = matcher.GetCacheStats()
	assert.Equal(t, 0, stats["total_patterns"])
}

func TestKeywordMatcher_ConcurrentAccess(t *testing.T) {
	matcher := NewKeywordMatcher()
	pattern := "^gpt-[0-9]+.*"
	modelName := "gpt-4-turbo"

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result := matcher.matchRegex(modelName, pattern)
			assert.True(t, result)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证缓存中只有一个编译后的正则表达式
	stats := matcher.GetCacheStats()
	assert.Equal(t, 1, stats["total_patterns"])
	assert.Equal(t, 1, stats["compiled_success"])
}