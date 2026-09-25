package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/middleware"
	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/service"
	"industrial-noise-source-attribution/backend/internal/util"
)

type MonitoringPointHandler struct {
	service *service.MonitoringPointService
}

func NewMonitoringPointHandler(value *service.MonitoringPointService) *MonitoringPointHandler {
	return &MonitoringPointHandler{service: value}
}

func (h *MonitoringPointHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	respond(c, http.StatusOK, result, err)
}

func (h *MonitoringPointHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}

func (h *MonitoringPointHandler) Create(c *gin.Context) {
	var request dto.CreateMonitoringPointRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	respond(c, http.StatusCreated, result, err)
}

func (h *MonitoringPointHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.UpdateMonitoringPointRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}

func (h *MonitoringPointHandler) Deactivate(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.AttributionActionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Deactivate(c.Request.Context(), id, request.Version, mustActor(c))
	respond(c, http.StatusOK, result, err)
}

func parseID(c *gin.Context, key string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil || value == 0 {
		util.RespondError(c, util.NewError(http.StatusBadRequest, "bad_request", key+" 必须是正整数", err))
		return 0, false
	}
	return uint(value), true
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		util.RespondError(c, util.NewError(http.StatusBadRequest, "validation_failed", "请求体格式或字段校验失败", err))
		return false
	}
	return true
}

func respond(c *gin.Context, status int, data any, err error) {
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, status, data)
}

func mustActor(c *gin.Context) model.Actor {
	actor, _ := middleware.ActorFromContext(c)
	return actor
}
