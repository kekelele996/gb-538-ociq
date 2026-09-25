package dto

import "time"

type CreateNoiseMeasurementRequest struct {
	MonitoringPointID uint               `json:"monitoring_point_id" binding:"required"`
	MeasuredAt        time.Time          `json:"measured_at" binding:"required"`
	DurationS         int                `json:"duration_s" binding:"required,gte=10,lte=86400"`
	OctaveBands       map[string]float64 `json:"octave_bands" binding:"required"`
	OverallDBA        float64            `json:"overall_dba" binding:"gte=0,lte=180"`
	BackgroundDBA     float64            `json:"background_dba" binding:"gte=0,lte=180"`
	WeatherNote       string             `json:"weather_note" binding:"max=500"`
	QualityReason     string             `json:"quality_reason" binding:"max=500"`
}

type MeasurementTransitionRequest struct {
	ToState string `json:"to_state" binding:"required"`
	Version uint   `json:"version" binding:"required,gte=1"`
}

type NoiseMeasurementResponse struct {
	ID                 uint               `json:"id"`
	MonitoringPointID  uint               `json:"monitoring_point_id"`
	PointCode          string             `json:"point_code"`
	PointName          string             `json:"point_name"`
	MeasuredAt         time.Time          `json:"measured_at"`
	DurationS          int                `json:"duration_s"`
	OctaveBands        map[string]float64 `json:"octave_bands"`
	NormalizedBands    map[string]float64 `json:"normalized_bands,omitempty"`
	OverallDBA         float64            `json:"overall_dba"`
	BackgroundDBA      float64            `json:"background_dba"`
	WeatherNote        string             `json:"weather_note"`
	SourceChecksum     string             `json:"source_checksum"`
	MeasurementQuality string             `json:"measurement_quality"`
	QualityReason      string             `json:"quality_reason"`
	MeasurementState   string             `json:"measurement_state"`
	ImportedBy         uint               `json:"imported_by"`
	Version            uint               `json:"version"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
