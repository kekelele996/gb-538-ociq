package model

import "time"

type MonitoringPoint struct {
	ID                    uint      `gorm:"primaryKey"`
	PointCode             string    `gorm:"size:64;uniqueIndex;not null"`
	Name                  string    `gorm:"size:160;not null"`
	XM                    float64   `gorm:"not null"`
	YM                    float64   `gorm:"not null"`
	HeightM               float64   `gorm:"not null"`
	AreaType              string    `gorm:"size:40;not null"`
	BackgroundProfileJSON string    `gorm:"type:text;not null"`
	OwnerTeam             string    `gorm:"size:120;not null"`
	PointState            string    `gorm:"size:24;index;not null"`
	Version               uint      `gorm:"not null;default:1"`
	CreatedAt             time.Time `gorm:"not null"`
	UpdatedAt             time.Time `gorm:"not null"`
}

func (MonitoringPoint) TableName() string { return "monitoring_points" }
