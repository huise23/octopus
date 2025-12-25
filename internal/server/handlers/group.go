package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bestruirui/octopus/internal/matcher"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/server/worker"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/group").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(getGroupList),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createGroup),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateGroup),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteGroup),
		).
		AddRoute(
			router.NewRoute("/auto-add-item", http.MethodPost).
				Handle(autoAddGroupItem),
		).
		AddRoute(
			router.NewRoute("/test-keywords", http.MethodPost).
				Handle(testKeywords),
		).
		AddRoute(
			router.NewRoute("/match-preview", http.MethodGet).
				Handle(matchPreview),
		)
}

func getGroupList(c *gin.Context) {
	groups, err := op.GroupList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, groups)
}

func createGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := op.GroupCreate(&group, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, group)
}

func updateGroup(c *gin.Context) {
	var req model.GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	group, err := op.GroupUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, group)
}

func deleteGroup(c *gin.Context) {
	id := c.Param("id")
	idNum, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := op.GroupDel(idNum, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, "group deleted successfully")
}

func autoAddGroupItem(c *gin.Context) {
	var req struct {
		ID int `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID <= 0 {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	err := worker.AutoAddGroupItem(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

// testKeywords 测试关键字匹配功能
func testKeywords(c *gin.Context) {
	var req struct {
		ModelName string                 `json:"model_name" binding:"required"`
		Keywords  []model.GroupKeyword   `json:"keywords" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 验证关键字配置
	keywordsJSON, err := json.Marshal(req.Keywords)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid keywords format")
		return
	}

	keywordMatcher := matcher.NewKeywordMatcher()
	if err := keywordMatcher.ValidateKeywords(string(keywordsJSON)); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid keywords: "+err.Error())
		return
	}

	// 测试每个关键字的匹配结果
	results := keywordMatcher.TestMatch(req.ModelName, req.Keywords)

	// 计算总体匹配结果
	overallMatch := false
	for _, result := range results {
		if result {
			overallMatch = true
			break
		}
	}

	resp.Success(c, gin.H{
		"model_name":     req.ModelName,
		"keywords":       req.Keywords,
		"match_results":  results,
		"overall_match":  overallMatch,
		"cache_stats":    keywordMatcher.GetCacheStats(),
	})
}

// matchPreview 预览匹配结果
func matchPreview(c *gin.Context) {
	// 获取查询参数
	modelName := c.Query("model_name")
	if modelName == "" {
		resp.Error(c, http.StatusBadRequest, "model_name is required")
		return
	}

	// 获取所有分组
	groups, err := op.GroupList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	keywordMatcher := matcher.NewKeywordMatcher()
	var matchedGroups []gin.H
	var unmatchedGroups []gin.H

	for _, group := range groups {
		matched := keywordMatcher.MatchModel(modelName, group)

		groupInfo := gin.H{
			"id":         group.ID,
			"name":       group.Name,
			"match_mode": group.MatchMode,
			"keywords":   group.Keywords,
		}

		// 解析关键字以便前端显示
		if group.Keywords != "" {
			var keywords []model.GroupKeyword
			if err := json.Unmarshal([]byte(group.Keywords), &keywords); err == nil {
				groupInfo["parsed_keywords"] = keywords
			}
		}

		if matched {
			// 添加匹配详情
			if group.MatchMode != model.GroupMatchModeNameOnly && group.Keywords != "" {
				var keywords []model.GroupKeyword
				if err := json.Unmarshal([]byte(group.Keywords), &keywords); err == nil {
					matchResults := keywordMatcher.TestMatch(modelName, keywords)
					groupInfo["keyword_match_details"] = matchResults
				}
			}
			matchedGroups = append(matchedGroups, groupInfo)
		} else {
			unmatchedGroups = append(unmatchedGroups, groupInfo)
		}
	}

	resp.Success(c, gin.H{
		"model_name":       modelName,
		"matched_groups":   matchedGroups,
		"unmatched_groups": unmatchedGroups,
		"total_groups":     len(groups),
		"matched_count":    len(matchedGroups),
		"cache_stats":      keywordMatcher.GetCacheStats(),
	})
}
