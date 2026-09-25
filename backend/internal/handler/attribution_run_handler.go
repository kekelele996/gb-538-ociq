package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/service"
)

type AttributionRunHandler struct {
	service *service.AttributionRunService
}

func NewAttributionRunHandler(value *service.AttributionRunService) *AttributionRunHandler {
	return &AttributionRunHandler{service: value}
}

func (h *AttributionRunHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	respond(c, http.StatusOK, result, err)
}

func (h *AttributionRunHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}

func (h *AttributionRunHandler) Create(c *gin.Context) {
	var request dto.CreateAttributionRunRequest
	if !bindJSON(c, &request) {
		return
	}
	result, reused, err := h.service.Create(c.Request.Context(), request, mustActor(c), c.GetHeader("Idempotency-Key"))
	status := http.StatusCreated
	if reused {
		status = http.StatusOK
		c.Header("X-Idempotent-Replay", "true")
	}
	respond(c, status, result, err)
}

func (h *AttributionRunHandler) Compare(c *gin.Context) {
	baseID, ok := parseID(c, "id")
	if !ok {
		return
	}
	otherID, ok := parseID(c, "other_id")
	if !ok {
		return
	}
	result, err := h.service.Compare(c.Request.Context(), baseID, otherID)
	respond(c, http.StatusOK, result, err)
}

func (h *AttributionRunHandler) Review(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.ReviewAttributionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Review(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}

func (h *AttributionRunHandler) Confirm(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.AttributionActionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Confirm(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}

func (h *AttributionRunHandler) Void(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.AttributionActionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Void(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
