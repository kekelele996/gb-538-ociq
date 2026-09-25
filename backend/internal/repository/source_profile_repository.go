package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"industrial-noise-source-attribution/backend/internal/model"
)

type SourceProfileRepository struct{ db *gorm.DB }

func NewSourceProfileRepository(db *gorm.DB) *SourceProfileRepository {
	return &SourceProfileRepository{db: db}
}

func (r *SourceProfileRepository) List(ctx context.Context) ([]model.SourceProfile, error) {
	var profiles []model.SourceProfile
	if err := r.db.WithContext(ctx).Order("source_code ASC, version DESC").Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("list source profiles: %w", err)
	}
	return profiles, nil
}

func (r *SourceProfileRepository) Get(ctx context.Context, id uint) (model.SourceProfile, error) {
	var profile model.SourceProfile
	if err := r.db.WithContext(ctx).First(&profile, id).Error; err != nil {
		return profile, fmt.Errorf("get source profile: %w", err)
	}
	return profile, nil
}

func (r *SourceProfileRepository) GetManyActive(ctx context.Context, ids []uint) ([]model.SourceProfile, error) {
	var profiles []model.SourceProfile
	if err := r.db.WithContext(ctx).Where("id IN ? AND profile_state = ?", ids, "active").Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("load active source profiles: %w", err)
	}
	if len(profiles) != len(ids) {
		return nil, gorm.ErrRecordNotFound
	}
	return profiles, nil
}

func (r *SourceProfileRepository) NextVersion(ctx context.Context, sourceCode string) (uint, error) {
	var maximum uint
	row := r.db.WithContext(ctx).Model(&model.SourceProfile{}).
		Where("source_code = ?", sourceCode).Select("COALESCE(MAX(version), 0)").Row()
	if err := row.Scan(&maximum); err != nil {
		return 0, fmt.Errorf("find next profile version: %w", err)
	}
	return maximum + 1, nil
}

func (r *SourceProfileRepository) Create(ctx context.Context, profile *model.SourceProfile, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(profile).Error; err != nil {
			return fmt.Errorf("create source profile: %w", err)
		}
		audit.EntityID = profile.ID
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit source profile creation: %w", err)
		}
		return nil
	})
}

func (r *SourceProfileRepository) Transition(ctx context.Context, id, expectedVersion uint, from, to string, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.SourceProfile{}).
			Where("id = ? AND lock_version = ? AND profile_state = ?", id, expectedVersion, from).
			Updates(map[string]any{"profile_state": to, "lock_version": gorm.Expr("lock_version + 1"), "updated_at": time.Now().UTC()})
		if result.Error != nil {
			return fmt.Errorf("transition source profile: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidData
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit source profile transition: %w", err)
		}
		return nil
	})
}
