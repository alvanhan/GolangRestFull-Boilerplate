package repository

import (
	"context"
	"time"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type AuditRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error)
	List(ctx context.Context, filter AuditFilter) ([]*entity.AuditLog, int64, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	GetByUser(ctx context.Context, userID uuid.UUID, filter AuditFilter) ([]*entity.AuditLog, int64, error)
	GetByResource(ctx context.Context, resourceID uuid.UUID, resourceType string, filter AuditFilter) ([]*entity.AuditLog, int64, error)
}

type AuditFilter struct {
	UserID       *uuid.UUID
	Action       *entity.AuditAction
	ResourceID   *uuid.UUID
	ResourceType *string
	IPAddress    string
	Status       string
	StartDate    *time.Time
	EndDate      *time.Time
	Page         int
	PageSize     int
	OrderBy      string
	OrderDir     string
}
