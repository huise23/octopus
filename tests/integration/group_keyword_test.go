package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/matcher"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/handlers"
	"github.com/bestruirui/octopus/internal/server/worker"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// GroupKeywordIntegrationTestSuite 分组关键字匹配集成测试套件
type GroupKeywordIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
	ctx    context.Context
}

func (suite *GroupKeywordIntegrationTestSuite) SetupSuite() {
	// 设置测试环境
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.ctx = context.Background()

	// 注册路由（这里需要根据实际的路由注册方式调整）
	// 假设有一个函数可以注册所有路由
	// handlers.RegisterRoutes(suite.router)
}

func (suite *GroupKeywordIntegrationTestSuite) SetupTest() {
	// 每个测试前的清理工作
	// 这里应该清理测试数据库
}

func (suite *GroupKeywordIntegrationTestSuite) TearDownTest() {
	// 每个测试后的清理工作
}

// TestKeywordMatcherIntegration 测试关键字匹配器的集成功能
func (suite *GroupKeywordIntegrationTestSuite) TestKeywordMatcherIntegration() {
	t := suite.T()

	// 创建测试分组
	testGroups := []model.Group{
		{
			Name:      "GPT模型",
			Mode:      model.GroupModeRoundRobin,
			Keywords:  `[{"pattern":"gpt-","type":"fuzzy"},{"pattern":"^gpt-[0-9]+.*","type":"regex"}]`,
			MatchMode: model.GroupMatchModeKeywordOnly,
		},
		{
			Name:      "Claude模型",
			Mode:      model.GroupModeRandom,
			Keywords:  `[{"pattern":"claude","type":"fuzzy"},{"pattern":"anthropic","type":"fuzzy"}]`,
			MatchMode: model.GroupMatchModeBoth,
		},
		{
			Name:      "传统分组",
			Mode:      model.GroupModeFailover,
			Keywords:  "",
			MatchMode: model.GroupMatchModeNameOnly,
		},
	}

	matcher := matcher.NewKeywordMatcher()

	// 测试各种模型名称的匹配
	testCases := []struct {
		modelName     string
		expectedMatch map[string]bool // 分组名称 -> 是否匹配
	}{
		{
			modelName: "gpt-4-turbo",
			expectedMatch: map[string]bool{
				"GPT模型":  true,
				"Claude模型": false,
				"传统分组":   false,
			},
		},
		{
			modelName: "claude-3-opus",
			expectedMatch: map[string]bool{
				"GPT模型":  false,
				"Claude模型": true,
				"传统分组":   false,
			},
		},
		{
			modelName: "claude-anthropic-model",
			expectedMatch: map[string]bool{
				"GPT模型":  false,
				"Claude模型": true, // 两个关键字都匹配
				"传统分组":   false,
			},
		},
		{
			modelName: "传统分组-test",
			expectedMatch: map[string]bool{
				"GPT模型":  false,
				"Claude模型": false,
				"传统分组":   true, // 分组名称匹配
			},
		},
	}

	for _, tc := range testCases {
		t.Run("测试模型: "+tc.modelName, func(t *testing.T) {
			for _, group := range testGroups {
				matched := matcher.MatchModel(tc.modelName, group)
				expected := tc.expectedMatch[group.Name]
				assert.Equal(t, expected, matched,
					"模型 %s 对分组 %s 的匹配结果不符合预期", tc.modelName, group.Name)
			}
		})
	}
}

// TestAutoGroupIntegration 测试自动分组集成功能
func (suite *GroupKeywordIntegrationTestSuite) TestAutoGroupIntegration() {
	t := suite.T()

	// 模拟创建渠道和分组
	// 这里需要根据实际的数据库操作进行调整

	// 测试自动分组功能
	testCases := []struct {
		name         string
		channelID    int
		channelName  string
		channelModel string
		customModel  string
		autoGroupType model.AutoGroupType
		expectedGroups []string // 期望被添加到的分组名称
	}{
		{
			name:         "GPT模型自动分组",
			channelID:    1,
			channelName:  "OpenAI",
			channelModel: "gpt-4,gpt-3.5-turbo",
			customModel:  "",
			autoGroupType: model.AutoGroupTypeFuzzy,
			expectedGroups: []string{"GPT模型"},
		},
		{
			name:         "Claude模型自动分组",
			channelID:    2,
			channelName:  "Anthropic",
			channelModel: "claude-3-opus,claude-3-sonnet",
			customModel:  "",
			autoGroupType: model.AutoGroupTypeFuzzy,
			expectedGroups: []string{"Claude模型"},
		},
		{
			name:         "混合模型自动分组",
			channelID:    3,
			channelName:  "Mixed",
			channelModel: "gpt-4",
			customModel:  "claude-3-haiku",
			autoGroupType: model.AutoGroupTypeFuzzy,
			expectedGroups: []string{"GPT模型", "Claude模型"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行自动分组
			worker.AutoGroup(tc.channelID, tc.channelName, tc.channelModel, tc.customModel, tc.autoGroupType)

			// 验证分组结果
			// 这里需要查询数据库验证 GroupItem 是否正确创建
			// 由于涉及数据库操作，这里只是示例结构
		})
	}
}

// TestChannelUpdateGroupItemCleanup 测试渠道更新时的分组项清理
func (suite *GroupKeywordIntegrationTestSuite) TestChannelUpdateGroupItemCleanup() {
	t := suite.T()

	// 创建测试渠道和分组项
	testChannel := &model.Channel{
		ID:          1,
		Name:        "TestChannel",
		Model:       "gpt-4,claude-3-opus",
		CustomModel: "custom-model",
		Enabled:     true,
	}

	// 模拟更新渠道，移除某些模型
	updatedChannel := &model.Channel{
		ID:          1,
		Name:        "TestChannel",
		Model:       "gpt-4", // 移除了 claude-3-opus
		CustomModel: "",      // 移除了 custom-model
		Enabled:     true,
	}

	// 执行渠道更新
	err := op.ChannelUpdate(updatedChannel, suite.ctx)
	assert.NoError(t, err, "渠道更新应该成功")

	// 验证分组项是否正确清理
	// 这里需要查询数据库验证被移除的模型对应的 GroupItem 是否被删除
	// 由于涉及数据库操作，这里只是示例结构
}

// TestAPIEndpoints 测试新增的API端点
func (suite *GroupKeywordIntegrationTestSuite) TestAPIEndpoints() {
	t := suite.T()

	// 测试关键字测试API
	suite.testKeywordsAPI(t)

	// 测试匹配预览API
	suite.testMatchPreviewAPI(t)
}

func (suite *GroupKeywordIntegrationTestSuite) testKeywordsAPI(t *testing.T) {
	// 准备测试数据
	testReq := map[string]interface{}{
		"model_name": "gpt-4-turbo",
		"keywords": []map[string]string{
			{"pattern": "gpt-4", "type": "fuzzy"},
			{"pattern": "^gpt-[0-9]+.*", "type": "regex"},
			{"pattern": "claude", "type": "fuzzy"},
		},
	}

	reqBody, _ := json.Marshal(testReq)
	req := httptest.NewRequest("POST", "/api/v1/group/test-keywords", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "API应该返回200状态码")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "响应应该是有效的JSON")

	// 验证响应结构
	assert.Contains(t, response, "model_name")
	assert.Contains(t, response, "keywords")
	assert.Contains(t, response, "match_results")
	assert.Contains(t, response, "overall_match")

	// 验证匹配结果
	matchResults := response["match_results"].([]interface{})
	assert.Len(t, matchResults, 3, "应该有3个匹配结果")
	assert.True(t, matchResults[0].(bool), "第一个关键字应该匹配")
	assert.True(t, matchResults[1].(bool), "第二个关键字应该匹配")
	assert.False(t, matchResults[2].(bool), "第三个关键字不应该匹配")
}

func (suite *GroupKeywordIntegrationTestSuite) testMatchPreviewAPI(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/group/match-preview?model_name=gpt-4-turbo", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "API应该返回200状态码")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "响应应该是有效的JSON")

	// 验证响应结构
	assert.Contains(t, response, "model_name")
	assert.Contains(t, response, "matched_groups")
	assert.Contains(t, response, "unmatched_groups")
	assert.Contains(t, response, "total_groups")
	assert.Contains(t, response, "matched_count")

	assert.Equal(t, "gpt-4-turbo", response["model_name"])
}

// TestPerformance 性能测试
func (suite *GroupKeywordIntegrationTestSuite) TestPerformance() {
	t := suite.T()

	matcher := matcher.NewKeywordMatcher()

	// 创建大量测试分组
	groups := make([]model.Group, 100)
	for i := 0; i < 100; i++ {
		groups[i] = model.Group{
			Name:      fmt.Sprintf("Group%d", i),
			Keywords:  `[{"pattern":"test-.*","type":"regex"},{"pattern":"model","type":"fuzzy"}]`,
			MatchMode: model.GroupMatchModeKeywordOnly,
		}
	}

	// 测试大量匹配操作的性能
	start := time.Now()
	for i := 0; i < 1000; i++ {
		modelName := fmt.Sprintf("test-model-%d", i)
		for _, group := range groups {
			matcher.MatchModel(modelName, group)
		}
	}
	duration := time.Since(start)

	t.Logf("1000次模型匹配操作耗时: %v", duration)
	assert.Less(t, duration, 5*time.Second, "性能测试应该在5秒内完成")

	// 测试正则表达式缓存效果
	stats := matcher.GetCacheStats()
	t.Logf("正则表达式缓存统计: %+v", stats)
	assert.Greater(t, stats["compiled_success"], 0, "应该有成功编译的正则表达式")
}

// TestConcurrency 并发测试
func (suite *GroupKeywordIntegrationTestSuite) TestConcurrency() {
	t := suite.T()

	matcher := matcher.NewKeywordMatcher()
	group := model.Group{
		Keywords:  `[{"pattern":"^test-[0-9]+.*","type":"regex"}]`,
		MatchMode: model.GroupMatchModeKeywordOnly,
	}

	// 并发执行匹配操作
	const numGoroutines = 50
	const numOperations = 100

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				modelName := fmt.Sprintf("test-%d-%d", id, j)
				result := matcher.MatchModel(modelName, group)
				assert.True(t, result, "模型应该匹配")
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证缓存状态
	stats := matcher.GetCacheStats()
	assert.Equal(t, 1, stats["total_patterns"], "应该只有一个正则表达式被缓存")
	assert.Equal(t, 1, stats["compiled_success"], "正则表达式应该编译成功")
}

// TestErrorHandling 错误处理测试
func (suite *GroupKeywordIntegrationTestSuite) TestErrorHandling() {
	t := suite.T()

	matcher := matcher.NewKeywordMatcher()

	// 测试无效的JSON格式
	group := model.Group{
		Keywords:  `invalid json`,
		MatchMode: model.GroupMatchModeKeywordOnly,
	}
	result := matcher.MatchModel("test-model", group)
	assert.False(t, result, "无效JSON应该返回false")

	// 测试无效的正则表达式
	group = model.Group{
		Keywords:  `[{"pattern":"[invalid-regex","type":"regex"}]`,
		MatchMode: model.GroupMatchModeKeywordOnly,
	}
	result = matcher.MatchModel("test-model", group)
	assert.False(t, result, "无效正则表达式应该返回false")

	// 测试关键字验证
	err := matcher.ValidateKeywords(`[{"pattern":"[invalid-regex","type":"regex"}]`)
	assert.Error(t, err, "应该检测到无效的正则表达式")

	err = matcher.ValidateKeywords(`invalid json`)
	assert.Error(t, err, "应该检测到无效的JSON格式")

	err = matcher.ValidateKeywords(`[]`)
	assert.NoError(t, err, "空数组应该是有效的")
}

// TestBackwardCompatibility 向后兼容性测试
func (suite *GroupKeywordIntegrationTestSuite) TestBackwardCompatibility() {
	t := suite.T()

	matcher := matcher.NewKeywordMatcher()

	// 测试传统的仅分组名称匹配模式
	group := model.Group{
		Name:      "gpt",
		Keywords:  "", // 空关键字
		MatchMode: model.GroupMatchModeNameOnly,
	}

	testCases := []struct {
		modelName string
		expected  bool
	}{
		{"gpt-4", true},
		{"gpt-3.5-turbo", true},
		{"claude-3", false},
		{"GPT-4", true}, // 大小写不敏感
	}

	for _, tc := range testCases {
		result := matcher.MatchModel(tc.modelName, group)
		assert.Equal(t, tc.expected, result,
			"传统匹配模式：模型 %s 的匹配结果应该是 %v", tc.modelName, tc.expected)
	}
}

// 运行集成测试套件
func TestGroupKeywordIntegrationSuite(t *testing.T) {
	suite.Run(t, new(GroupKeywordIntegrationTestSuite))
}

// BenchmarkKeywordMatching 关键字匹配性能基准测试
func BenchmarkKeywordMatching(b *testing.B) {
	matcher := matcher.NewKeywordMatcher()
	group := model.Group{
		Keywords:  `[{"pattern":"gpt-","type":"fuzzy"},{"pattern":"^gpt-[0-9]+.*","type":"regex"}]`,
		MatchMode: model.GroupMatchModeKeywordOnly,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.MatchModel("gpt-4-turbo", group)
	}
}

// BenchmarkRegexCompilation 正则表达式编译性能基准测试
func BenchmarkRegexCompilation(b *testing.B) {
	matcher := matcher.NewKeywordMatcher()
	patterns := []string{
		"^gpt-[0-9]+.*",
		"claude-[0-9]+-.*",
		"gemini-pro-.*",
		"^anthropic-.*",
		"openai-.*-model$",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := patterns[i%len(patterns)]
		group := model.Group{
			Keywords:  fmt.Sprintf(`[{"pattern":"%s","type":"regex"}]`, pattern),
			MatchMode: model.GroupMatchModeKeywordOnly,
		}
		matcher.MatchModel("test-model", group)
	}
}