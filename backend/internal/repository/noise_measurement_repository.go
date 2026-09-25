package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"industrial-noise-source-attribution/backend/internal/model"
)

type NoiseMeasurementRepository struct{ db *gorm.DB }

func NewNoiseMeasurementRepository(db *gorm.DB) *NoiseMeasurementRepository {
	return &NoiseMeasurementRepository{db: db}
}

func (r *NoiseMeasurementRepository) List(ctx context.Context) ([]model.NoiseMeasurement, error) {
	var measurements []model.NoiseMeasurement
	if err := r.db.WithContext(ctx).Preload("MonitoringPoint").
		Order("measured_at DESC, id DESC").Find(&measurements).Error; err != nil {
		return nil, fmt.Errorf("list noise measurements: %w", err)
	}
	return measurements, nil
}

func (r *NoiseMeasurementRepository) Get(ctx context.Context, id uint) (model.NoiseMeasurement, error) {
	var measurement model.NoiseMeasurement
	if err := r.db.WithContext(ctx).Preload("MonitoringPoint").First(&measurement, id).Error; err != nil {
		return measurement, fmt.Errorf("get noise measurement: %w", err)
	}
	return measurement, nil
}

func (r *NoiseMeasurementRepository) GetManyReady(ctx context.Context, ids []uint) ([]model.NoiseMeasurement, error) {
	var measurements []model.NoiseMeasurement
	if err := r.db.WithContext(ctx).Preload("MonitoringPoint").
		Where("id IN ? AND measurement_state = ?", ids, "ready").Find(&measurements).Error; err != nil {
		return nil, fmt.Errorf("load ready noise measurements: %w", err)
	}
	if len(measurements) != len(ids) {
		return nil, gorm.ErrRecordNotFound
	}
	return measurements, nil
}

func (r *NoiseMeasurementRepository) FindByChecksum(ctx context.Context, pointID uint, checksum string) (model.NoiseMeasurement, error) {
	var measurement model.NoiseMeasurement
	err := r.db.WithContext(ctx).Preload("MonitoringPoint").
		Where("monitoring_point_id = ? AND source_checksum = ?", pointID, checksum).
		Order("id DESC").First(&measurement).Error
	if err != nil {
		return measurement, fmt.Errorf("find measurement checksum: %w", err)
	}
	return measurement, nil
}

func (r *NoiseMeasurementRepository) Create(ctx context.Context, measurement *model.NoiseMeasurement, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(measurement).Error; err != nil {
			return fmt.Errorf("create noise measurement: %w", err)
		}
		audit.EntityID = measurement.ID
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit measurement import: %w", err)
		}
		return nil
	})
}

func (r *NoiseMeasurementRepository) Transition(ctx context.Context, id, expectedVersion uint, from, to, quality, reason string, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"measurement_state": to, "version": gorm.Expr("version + 1"), "updated_at": time.Now().UTC(),
		}
		if quality != "" {
			updates["measurement_quality"] = quality
			updates["quality_reason"] = reason
		}
		result := tx.Model(&model.NoiseMeasurement{}).
			Where("id = ? AND version = ? AND measurement_state = ?", id, expectedVersion, from).
			Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition noise measurement: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidData
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit measurement transition: %w", err)
		}
		return nil
	})
}
