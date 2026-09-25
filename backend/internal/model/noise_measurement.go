package model

import "time"

type NoiseMeasurement struct {
	ID                 uint `gorm:"primaryKey"`
	MonitoringPointID  uint `gorm:"index;uniqueIndex:idx_measurement_point_checksum;not null"`
	MonitoringPoint    MonitoringPoint
	MeasuredAt         time.Time `gorm:"index;not null"`
	DurationS          int       `gorm:"not null"`
	OctaveBandsJSON    string    `gorm:"type:text;not null"`
	OverallDBA         float64   `gorm:"not null"`
	BackgroundDBA      float64   `gorm:"not null"`
	WeatherNote        string    `gorm:"size:500"`
	SourceChecksum     string    `gorm:"size:64;index;uniqueIndex:idx_measurement_point_checksum;not null"`
	MeasurementQuality string    `gorm:"size:24;index;not null"`
	QualityReason      string    `gorm:"size:500"`
	MeasurementState   string    `gorm:"size:24;index;not null"`
	ImportedBy         uint      `gorm:"index;not null"`
	Version            uint      `gorm:"not null;default:1"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (NoiseMeasurement) TableName() string { return "noise_measurements" }
