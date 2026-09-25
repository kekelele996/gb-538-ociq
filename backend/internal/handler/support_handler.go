package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/service"
	"industrial-noise-source-attribution/backend/internal/util"
)

type SupportHandler struct{ service *service.AccessService }

func NewSupportHandler(value *service.AccessService) *SupportHandler {
	return &SupportHandler{service: value}
}

func (h *SupportHandler) Login(c *gin.Context) {
	var request service.LoginRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Login(c.Request.Context(), request)
	respond(c, http.StatusOK, result, err)
}

func (h *SupportHandler) Audits(c *gin.Context) {
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 500 {
			util.RespondError(c, util.NewError(http.StatusBadRequest, "bad_request", "limit 必须在 1 到 500 之间", err))
			return
		}
		limit = value
	}
	result, err := h.service.ListAudits(c.Request.Context(), repository.AuditFilter{
		EntityType: c.Query("entity_type"), RequestID: c.Query("request_id"), Actor: c.Query("actor"), Limit: limit,
	})
	respond(c, http.StatusOK, result, err)
}

func (h *SupportHandler) Enums(c *gin.Context) {
	util.Respond(c, http.StatusOK, gin.H{
		"octave_bands": constants.OctaveBands, "measurement_qualities": constants.MeasurementQualities,
		"measurement_states": []constants.MeasurementState{
			constants.MeasurementCaptured, constants.MeasurementValidated, constants.MeasurementNormalized,
			constants.MeasurementReady, constants.MeasurementRejected, constants.MeasurementSuperseded,
		},
		"attribution_states": []constants.AttributionState{
			constants.AttributionQueued, constants.AttributionCalculating, constants.AttributionCompleted,
			constants.AttributionFailed, constants.AttributionReviewed, constants.AttributionConfirmed, constants.AttributionVoided,
		},
		"roles": constants.Roles, "algorithm_version": constants.AlgorithmVersion,
	})
}
