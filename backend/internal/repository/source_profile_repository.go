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

// FindActiveBySourceCode 返回同一设备当前启用中的版本（排除自身），不存在时返回 gorm.ErrRecordNotFound。
func (r *SourceProfileRepository) FindActiveBySourceCode(ctx context.Context, sourceCode string, excludeID uint) (model.SourceProfile, error) {
	var profile model.SourceProfile
	if err := r.db.WithContext(ctx).
		Where("source_code = ? AND profile_state = ? AND id <> ?", sourceCode, "active", excludeID).
		First(&profile).Error; err != nil {
		return profile, fmt.Errorf("find active source profile for %s: %w", sourceCode, err)
	}
	return profile, nil
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

// Activate 在同一事务中启用目标版本并退出同设备当前启用版本。
// 两个条件更新都基于 id + lock_version + 当前状态，任一步影响行数不为 1 即整体回滚，
// 并发下只有一方能提交；数据库唯一索引兜底无启用版本可读时的并发激活。
func (r *SourceProfileRepository) Activate(ctx context.Context, targetID, targetLock uint, from string, superseded *model.SourceProfile, audits ...*model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if superseded != nil {
			result := tx.Model(&model.SourceProfile{}).
				Where("id = ? AND lock_version = ? AND profile_state = ?", superseded.ID, superseded.LockVersion, "active").
				Updates(map[string]any{"profile_state": "retired", "lock_version": gorm.Expr("lock_version + 1"), "updated_at": now})
			if result.Error != nil {
				return fmt.Errorf("retire superseded source profile: %w", result.Error)
			}
			if result.RowsAffected != 1 {
				return gorm.ErrInvalidData
			}
		}
		result := tx.Model(&model.SourceProfile{}).
			Where("id = ? AND lock_version = ? AND profile_state = ?", targetID, targetLock, from).
			Updates(map[string]any{"profile_state": "active", "lock_version": gorm.Expr("lock_version + 1"), "updated_at": now})
		if result.Error != nil {
			return fmt.Errorf("activate source profile: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidData
		}
		for _, audit := range audits {
			if err := tx.Create(audit).Error; err != nil {
				return fmt.Errorf("audit source profile activation: %w", err)
			}
		}
		return nil
	})
}

// SingleActiveIndexDDL 保证同一 source_code 至多一个启用版本，PostgreSQL 与 SQLite 均支持部分索引。
const SingleActiveIndexDDL = "CREATE UNIQUE INDEX IF NOT EXISTS idx_source_profiles_single_active ON source_profiles (source_code) WHERE profile_state = 'active'"

func (r *SourceProfileRepository) EnsureSingleActiveIndex(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Exec(SingleActiveIndexDDL).Error; err != nil {
		return fmt.Errorf("ensure single active source profile index: %w", err)
	}
	return nil
}

// RetireDuplicateActives 在建立唯一索引前收敛历史数据：同一设备只保留最高版本启用，其余退出候选。
func (r *SourceProfileRepository) RetireDuplicateActives(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actives []model.SourceProfile
		if err := tx.Where("profile_state = ?", "active").Order("source_code ASC, version DESC").Find(&actives).Error; err != nil {
			return fmt.Errorf("list duplicated active source profiles: %w", err)
		}
		seen := make(map[string]bool, len(actives))
		for _, profile := range actives {
			if seen[profile.SourceCode] {
				result := tx.Model(&model.SourceProfile{}).Where("id = ?", profile.ID).
					Updates(map[string]any{"profile_state": "retired", "lock_version": gorm.Expr("lock_version + 1"), "updated_at": time.Now().UTC()})
				if result.Error != nil {
					return fmt.Errorf("retire duplicated active source profile: %w", result.Error)
				}
				continue
			}
			seen[profile.SourceCode] = true
		}
		return nil
	})
}
