package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"industrial-noise-source-attribution/backend/internal/model"
)

type AuditFilter struct {
	EntityType string
	RequestID  string
	Actor      string
	Limit      int
}

type SupportRepository struct{ db *gorm.DB }

func NewSupportRepository(db *gorm.DB) *SupportRepository { return &SupportRepository{db: db} }

func (r *SupportRepository) FindUserByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ? AND active = ?", username, true).First(&user).Error; err != nil {
		return user, fmt.Errorf("find active user: %w", err)
	}
	return user, nil
}

func (r *SupportRepository) ListAudits(ctx context.Context, filter AuditFilter) ([]model.AuditLog, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}
	if filter.Actor != "" {
		query = query.Where("actor_name LIKE ?", "%"+filter.Actor+"%")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var logs []model.AuditLog
	if err := query.Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, nil
}

func IsNotFound(err error) bool { return err != nil && gorm.ErrRecordNotFound == rootGormError(err) }

func rootGormError(err error) error {
	for err != nil {
		if err == gorm.ErrRecordNotFound || err == gorm.ErrInvalidData {
			return err
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = unwrapper.Unwrap()
	}
	return err
}

func IsConflict(err error) bool { return rootGormError(err) == gorm.ErrInvalidData }
