package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// FindActiveSibling 返回同一声源编号当前启用中的其他版本；没有时返回 gorm.ErrRecordNotFound。
func (r *SourceProfileRepository) FindActiveSibling(ctx context.Context, sourceCode string, excludeID uint) (model.SourceProfile, error) {
	var profile model.SourceProfile
	err := r.db.WithContext(ctx).
		Where("source_code = ? AND profile_state = ? AND id <> ?", sourceCode, "active", excludeID).
		First(&profile).Error
	if err != nil {
		return profile, fmt.Errorf("find active sibling source profile: %w", err)
	}
	return profile, nil
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
			Updates(map[string]any{"profile_state": to, "lock_version": gorm.Expr("lock_version + 1"), "superseded_by_id": nil, "updated_at": time.Now().UTC()})
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

// Activate 在同一事务中启用目标版本，并让同一声源编号当前启用的旧版本退出候选。
// expectedActiveID 为 nil 表示调用方认为当前没有启用版本；任何校验失败都会整体回滚，
// 两个版本都不会留下半套状态。PostgreSQL 下先锁定整个版本组以序列化并发启用。
func (r *SourceProfileRepository) Activate(ctx context.Context, targetID, expectedLockVersion uint, expectedActiveID *uint, activateAudit, supersedeAudit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sourceCode := tx.Model(&model.SourceProfile{}).Select("source_code").Where("id = ?", targetID)
		if tx.Dialector.Name() == "postgres" {
			var locked []uint
			if err := tx.Model(&model.SourceProfile{}).Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("source_code = (?)", sourceCode).Pluck("id", &locked).Error; err != nil {
				return fmt.Errorf("lock source profile group: %w", err)
			}
		}
		now := time.Now().UTC()
		activated := tx.Model(&model.SourceProfile{}).
			Where("id = ? AND lock_version = ? AND profile_state IN ?", targetID, expectedLockVersion, []string{"draft", "retired"}).
			Updates(map[string]any{
				"profile_state": "active", "lock_version": gorm.Expr("lock_version + 1"),
				"superseded_by_id": nil, "updated_at": now,
			})
		if activated.Error != nil {
			return fmt.Errorf("activate source profile: %w", activated.Error)
		}
		if activated.RowsAffected != 1 {
			return gorm.ErrInvalidData
		}
		var siblings []model.SourceProfile
		if err := tx.Where("source_code = (?) AND profile_state = ? AND id <> ?", sourceCode, "active", targetID).
			Find(&siblings).Error; err != nil {
			return fmt.Errorf("check active sibling source profile: %w", err)
		}
		if expectedActiveID == nil {
			if len(siblings) != 0 {
				return gorm.ErrInvalidData
			}
		} else {
			if len(siblings) != 1 || siblings[0].ID != *expectedActiveID {
				return gorm.ErrInvalidData
			}
			retired := tx.Model(&model.SourceProfile{}).
				Where("id = ? AND profile_state = ?", *expectedActiveID, "active").
				Updates(map[string]any{
					"profile_state": "retired", "lock_version": gorm.Expr("lock_version + 1"),
					"superseded_by_id": targetID, "updated_at": now,
				})
			if retired.Error != nil {
				return fmt.Errorf("supersede source profile: %w", retired.Error)
			}
			if retired.RowsAffected != 1 {
				return gorm.ErrInvalidData
			}
		}
		if err := tx.Create(activateAudit).Error; err != nil {
			return fmt.Errorf("audit source profile activation: %w", err)
		}
		if supersedeAudit != nil {
			if err := tx.Create(supersedeAudit).Error; err != nil {
				return fmt.Errorf("audit source profile supersede: %w", err)
			}
		}
		return nil
	})
}
