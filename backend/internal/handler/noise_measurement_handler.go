package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/service"
)

type NoiseMeasurementHandler struct {
	service *service.NoiseMeasurementService
}

func NewNoiseMeasurementHandler(value *service.NoiseMeasurementService) *NoiseMeasurementHandler {
	return &NoiseMeasurementHandler{service: value}
}

func (h *NoiseMeasurementHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	respond(c, http.StatusOK, result, err)
}

func (h *NoiseMeasurementHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}

func (h *NoiseMeasurementHandler) Create(c *gin.Context) {
	var request dto.CreateNoiseMeasurementRequest
	if !bindJSON(c, &request) {
		return
	}
	result, reused, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	status := http.StatusCreated
	if reused {
		status = http.StatusOK
		c.Header("X-Idempotent-Replay", "true")
	}
	respond(c, status, result, err)
}

func (h *NoiseMeasurementHandler) Transition(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request dto.MeasurementTransitionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Transition(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
