package dto

import "time"

type CreateSourceProfileRequest struct {
	SourceCode         string             `json:"source_code" binding:"required,min=2,max=64"`
	Name               string             `json:"name" binding:"required,min=2,max=160"`
	XM                 float64            `json:"x_m" binding:"gte=-100000,lte=100000"`
	YM                 float64            `json:"y_m" binding:"gte=-100000,lte=100000"`
	HeightM            float64            `json:"height_m" binding:"gte=0,lte=100"`
	ReferenceDistanceM float64            `json:"reference_distance_m" binding:"required,gt=0,lte=1000"`
	OctavePower        map[string]float64 `json:"octave_power" binding:"required"`
	Directivity        map[string]float64 `json:"directivity" binding:"required"`
	OperatingFactor    float64            `json:"operating_factor" binding:"gt=0,lte=1"`
}

type ProfileTransitionRequest struct {
	ToState     string `json:"to_state" binding:"required"`
	LockVersion uint   `json:"lock_version" binding:"required,gte=1"`
}

type SourceProfileResponse struct {
	ID                 uint               `json:"id"`
	SourceCode         string             `json:"source_code"`
	Name               string             `json:"name"`
	XM                 float64            `json:"x_m"`
	YM                 float64            `json:"y_m"`
	HeightM            float64            `json:"height_m"`
	ReferenceDistanceM float64            `json:"reference_distance_m"`
	OctavePower        map[string]float64 `json:"octave_power"`
	Directivity        map[string]float64 `json:"directivity"`
	OperatingFactor    float64            `json:"operating_factor"`
	ProfileState       string             `json:"profile_state"`
	Version            uint               `json:"version"`
	LockVersion        uint               `json:"lock_version"`
	CreatedBy          uint               `json:"created_by"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
