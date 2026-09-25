package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"industrial-noise-source-attribution/backend/internal/model"
)

type AttributionRunRepository struct{ db *gorm.DB }

func NewAttributionRunRepository(db *gorm.DB) *AttributionRunRepository {
	return &AttributionRunRepository{db: db}
}

func (r *AttributionRunRepository) List(ctx context.Context) ([]model.AttributionRun, error) {
	var runs []model.AttributionRun
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("list attribution runs: %w", err)
	}
	return runs, nil
}

func (r *AttributionRunRepository) Get(ctx context.Context, id uint) (model.AttributionRun, error) {
	var run model.AttributionRun
	if err := r.db.WithContext(ctx).First(&run, id).Error; err != nil {
		return run, fmt.Errorf("get attribution run: %w", err)
	}
	return run, nil
}

func (r *AttributionRunRepository) FindByInput(ctx context.Context, hash, version string) (model.AttributionRun, error) {
	var run model.AttributionRun
	if err := r.db.WithContext(ctx).Where("input_hash = ? AND algorithm_version = ?", hash, version).
		First(&run).Error; err != nil {
		return run, fmt.Errorf("find attribution input: %w", err)
	}
	return run, nil
}

func (r *AttributionRunRepository) CreateCalculated(ctx context.Context, run *model.AttributionRun, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		finalState := run.AttributionState
		finishedAt := run.FinishedAt
		run.AttributionState = "queued"
		run.FinishedAt = nil
		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("create queued attribution run: %w", err)
		}
		calculating := tx.Model(&model.AttributionRun{}).Where("id = ? AND attribution_state = ?", run.ID, "queued").
			Updates(map[string]any{"attribution_state": "calculating", "version": gorm.Expr("version + 1")})
		if calculating.Error != nil || calculating.RowsAffected != 1 {
			return fmt.Errorf("start attribution calculation: %w", calculating.Error)
		}
		completed := tx.Model(&model.AttributionRun{}).Where("id = ? AND attribution_state = ?", run.ID, "calculating").
			Updates(map[string]any{
				"attribution_state": finalState, "finished_at": finishedAt,
				"normalized_bands_json": run.NormalizedBandsJSON, "contributions_json": run.ContributionsJSON,
				"evidence_json": run.EvidenceJSON, "residual_error": run.ResidualError,
				"explanation": run.Explanation, "version": gorm.Expr("version + 1"), "updated_at": time.Now().UTC(),
			})
		if completed.Error != nil || completed.RowsAffected != 1 {
			return fmt.Errorf("complete attribution calculation: %w", completed.Error)
		}
		run.AttributionState = finalState
		run.FinishedAt = finishedAt
		run.Version += 2
		audit.EntityID = run.ID
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit attribution calculation: %w", err)
		}
		return nil
	})
}

func (r *AttributionRunRepository) Transition(ctx context.Context, id, expectedVersion uint, from, to string, reviewedBy *uint, reviewNote string, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"attribution_state": to, "version": gorm.Expr("version + 1"), "updated_at": time.Now().UTC(),
		}
		if reviewedBy != nil {
			updates["reviewed_by"] = *reviewedBy
		}
		if reviewNote != "" {
			updates["review_note"] = reviewNote
		}
		result := tx.Model(&model.AttributionRun{}).
			Where("id = ? AND version = ? AND attribution_state = ?", id, expectedVersion, from).
			Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition attribution run: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidData
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("audit attribution transition: %w", err)
		}
		return nil
	})
}
