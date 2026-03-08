package repository

import (
	"context"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
	GetByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*entity.Notification, int64, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}
