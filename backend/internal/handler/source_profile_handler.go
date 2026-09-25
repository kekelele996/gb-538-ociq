package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/service"
)

type SourceProfileHandler struct{ service *service.SourceProfileService }

func NewSourceProfileHandler(value *service.SourceProfileService) *SourceProfileHandler {
	return &SourceProfileHandler{service: value}
}

func (h *SourceProfileHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	respond(c, http.StatusOK, result, err)
}

func (h *SourceProfileHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}

func (h *SourceProfileHandler) Create(c *gin.Context) {
	var request dto.CreateSourceProfileRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	respond(c, http.StatusCreated, result, err)
}

func (h *SourceProfileHandler) Transition(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.ProfileTransitionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Transition(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
