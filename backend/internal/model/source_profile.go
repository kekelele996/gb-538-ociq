package model

import "time"

type SourceProfile struct {
	ID                 uint      `gorm:"primaryKey"`
	SourceCode         string    `gorm:"size:64;not null;uniqueIndex:idx_source_version"`
	Name               string    `gorm:"size:160;not null"`
	XM                 float64   `gorm:"not null"`
	YM                 float64   `gorm:"not null"`
	HeightM            float64   `gorm:"not null"`
	ReferenceDistanceM float64   `gorm:"not null"`
	OctavePowerJSON    string    `gorm:"type:text;not null"`
	DirectivityJSON    string    `gorm:"type:text;not null"`
	OperatingFactor    float64   `gorm:"not null"`
	ProfileState       string    `gorm:"size:24;index;not null"`
	Version            uint      `gorm:"not null;uniqueIndex:idx_source_version"`
	LockVersion        uint      `gorm:"not null;default:1"`
	CreatedBy          uint      `gorm:"index;not null"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (SourceProfile) TableName() string { return "source_profiles" }
