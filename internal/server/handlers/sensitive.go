package handlers

import (
	"net/http"
	"strconv"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/sensitive").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listSensitiveRules),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(createSensitiveRule),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(updateSensitiveRule),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteSensitiveRule),
		).
		AddRoute(
			router.NewRoute("/toggle/:id", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(toggleSensitiveRule),
		)
}

func listSensitiveRules(c *gin.Context) {
	rules, err := op.SensitiveFilterRuleList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rules)
}

func createSensitiveRule(c *gin.Context) {
	var rule model.SensitiveFilterRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.SensitiveFilterRuleCreate(&rule, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rule)
}

func updateSensitiveRule(c *gin.Context) {
	var rule model.SensitiveFilterRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.SensitiveFilterRuleUpdate(&rule, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rule)
}

func deleteSensitiveRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.SensitiveFilterRuleDelete(id, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

type toggleRequest struct {
	Enabled bool `json:"enabled"`
}

func toggleSensitiveRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	var req toggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.SensitiveFilterRuleToggle(id, req.Enabled, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}
