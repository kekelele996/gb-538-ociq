package dto

import "time"

type CreateMonitoringPointRequest struct {
	PointCode         string             `json:"point_code" binding:"required,min=2,max=64"`
	Name              string             `json:"name" binding:"required,min=2,max=160"`
	XM                float64            `json:"x_m" binding:"gte=-100000,lte=100000"`
	YM                float64            `json:"y_m" binding:"gte=-100000,lte=100000"`
	HeightM           float64            `json:"height_m" binding:"gte=0,lte=100"`
	AreaType          string             `json:"area_type" binding:"required,oneof=boundary workshop office residential"`
	BackgroundProfile map[string]float64 `json:"background_profile" binding:"required"`
	OwnerTeam         string             `json:"owner_team" binding:"required,max=120"`
}

type UpdateMonitoringPointRequest struct {
	Name              string             `json:"name" binding:"required,min=2,max=160"`
	XM                float64            `json:"x_m" binding:"gte=-100000,lte=100000"`
	YM                float64            `json:"y_m" binding:"gte=-100000,lte=100000"`
	HeightM           float64            `json:"height_m" binding:"gte=0,lte=100"`
	AreaType          string             `json:"area_type" binding:"required,oneof=boundary workshop office residential"`
	BackgroundProfile map[string]float64 `json:"background_profile" binding:"required"`
	OwnerTeam         string             `json:"owner_team" binding:"required,max=120"`
	Version           uint               `json:"version" binding:"required,gte=1"`
}

type MonitoringPointResponse struct {
	ID                    uint               `json:"id"`
	PointCode             string             `json:"point_code"`
	Name                  string             `json:"name"`
	XM                    float64            `json:"x_m"`
	YM                    float64            `json:"y_m"`
	HeightM               float64            `json:"height_m"`
	AreaType              string             `json:"area_type"`
	BackgroundProfile     map[string]float64 `json:"background_profile"`
	OwnerTeam             string             `json:"owner_team"`
	PointState            string             `json:"point_state"`
	Version               uint               `json:"version"`
	MeasurementCount      int64              `json:"measurement_count"`
	LatestQuality         string             `json:"latest_quality"`
	LatestMeasurementTime *time.Time         `json:"latest_measurement_time"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
}
