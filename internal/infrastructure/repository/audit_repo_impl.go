package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"file-management-service/internal/domain/entity"
	domainerrors "file-management-service/internal/domain/errors"
	domrepo "file-management-service/internal/domain/repository"
)

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) domrepo.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create audit log", err)
	}
	return nil
}

func (r *auditRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	var log entity.AuditLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("audit log")
		}
		return nil, domainerrors.Wrap(500, "failed to get audit log by id", err)
	}
	return &log, nil
}

func (r *auditRepository) List(ctx context.Context, filter domrepo.AuditFilter) ([]*entity.AuditLog, int64, error) {
	query := r.applyFilter(r.db.WithContext(ctx).Model(&entity.AuditLog{}), filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count audit logs", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var logs []*entity.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to list audit logs", err)
	}
	return logs, total, nil
}

func (r *auditRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&entity.AuditLog{})
	if result.Error != nil {
		return 0, domainerrors.Wrap(500, "failed to delete old audit logs", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *auditRepository) GetByUser(ctx context.Context, userID uuid.UUID, filter domrepo.AuditFilter) ([]*entity.AuditLog, int64, error) {
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&entity.AuditLog{}).Where("user_id = ?", userID),
		filter,
	)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count audit logs by user", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var logs []*entity.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to get audit logs by user", err)
	}
	return logs, total, nil
}

func (r *auditRepository) GetByResource(ctx context.Context, resourceID uuid.UUID, resourceType string, filter domrepo.AuditFilter) ([]*entity.AuditLog, int64, error) {
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&entity.AuditLog{}).
			Where("resource_id = ? AND resource_type = ?", resourceID, resourceType),
		filter,
	)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count audit logs by resource", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var logs []*entity.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to get audit logs by resource", err)
	}
	return logs, total, nil
}

func (r *auditRepository) applyFilter(query *gorm.DB, f domrepo.AuditFilter) *gorm.DB {
	if f.UserID != nil {
		query = query.Where("user_id = ?", *f.UserID)
	}
	if f.Action != nil {
		query = query.Where("action = ?", *f.Action)
	}
	if f.ResourceID != nil {
		query = query.Where("resource_id = ?", *f.ResourceID)
	}
	if f.ResourceType != nil {
		query = query.Where("resource_type = ?", *f.ResourceType)
	}
	if f.IPAddress != "" {
		query = query.Where("ip_address = ?", f.IPAddress)
	}
	if f.Status != "" {
		query = query.Where("status = ?", f.Status)
	}
	if f.StartDate != nil {
		query = query.Where("created_at >= ?", *f.StartDate)
	}
	if f.EndDate != nil {
		query = query.Where("created_at <= ?", *f.EndDate)
	}
	return query
}
