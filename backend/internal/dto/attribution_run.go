package dto

import "time"

type CreateAttributionRunRequest struct {
	MeasurementIDs   []uint `json:"measurement_ids" binding:"required,min=1,dive,gt=0"`
	SourceProfileIDs []uint `json:"source_profile_ids" binding:"required,min=1,dive,gt=0"`
}

type ReviewAttributionRequest struct {
	Note    string `json:"note" binding:"required,min=3,max=1000"`
	Version uint   `json:"version" binding:"required,gte=1"`
}

type AttributionActionRequest struct {
	Version uint `json:"version" binding:"required,gte=1"`
}

type BandContribution struct {
	BandHz      int     `json:"band_hz"`
	PredictedDB float64 `json:"predicted_db"`
	EnergyShare float64 `json:"energy_share"`
}

type SourceContribution struct {
	SourceProfileID uint               `json:"source_profile_id"`
	SourceCode      string             `json:"source_code"`
	SourceName      string             `json:"source_name"`
	Coefficient     float64            `json:"coefficient"`
	ContributionPct float64            `json:"contribution_pct"`
	OverallDB       float64            `json:"overall_db"`
	Bands           []BandContribution `json:"bands"`
}

type AttributionEvidence struct {
	MatrixRows      int      `json:"matrix_rows"`
	MatrixColumns   int      `json:"matrix_columns"`
	Iterations      int      `json:"iterations"`
	Converged       bool     `json:"converged"`
	ConditionHint   float64  `json:"condition_hint"`
	UnreliableBands []string `json:"unreliable_bands"`
	Warnings        []string `json:"warnings"`
	Objective       float64  `json:"objective"`
	ElapsedMillis   int64    `json:"elapsed_millis"`
}

type AttributionRunResponse struct {
	ID               uint                 `json:"id"`
	RunCode          string               `json:"run_code"`
	MeasurementIDs   []uint               `json:"measurement_ids"`
	SourceProfileIDs []uint               `json:"source_profile_ids"`
	AlgorithmVersion string               `json:"algorithm_version"`
	InputHash        string               `json:"input_hash"`
	InputSnapshot    any                  `json:"input_snapshot"`
	NormalizedBands  any                  `json:"normalized_bands"`
	Contributions    []SourceContribution `json:"contributions"`
	Evidence         AttributionEvidence  `json:"evidence"`
	ResidualError    float64              `json:"residual_error"`
	AttributionState string               `json:"attribution_state"`
	Explanation      string               `json:"explanation"`
	StartedAt        time.Time            `json:"started_at"`
	FinishedAt       *time.Time           `json:"finished_at"`
	CreatedBy        uint                 `json:"created_by"`
	ReviewedBy       *uint                `json:"reviewed_by"`
	ReviewNote       string               `json:"review_note"`
	Version          uint                 `json:"version"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

type AttributionComparisonResponse struct {
	BaseRunID        uint    `json:"base_run_id"`
	OtherRunID       uint    `json:"other_run_id"`
	ResidualDelta    float64 `json:"residual_delta"`
	TopSourceChanged bool    `json:"top_source_changed"`
	BaseTopSource    string  `json:"base_top_source"`
	OtherTopSource   string  `json:"other_top_source"`
	Explanation      string  `json:"explanation"`
}
