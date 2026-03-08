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

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) domrepo.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *entity.Notification) error {
	if err := r.db.WithContext(ctx).Create(notification).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create notification", err)
	}
	return nil
}

func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	var notif entity.Notification
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&notif).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("notification")
		}
		return nil, domainerrors.Wrap(500, "failed to get notification by id", err)
	}
	return &notif, nil
}

func (r *notificationRepository) GetByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*entity.Notification, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count notifications", err)
	}

	query = query.Order("created_at DESC")

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	var notifications []*entity.Notification
	if err := query.Find(&notifications).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to get notifications by user", err)
	}
	return notifications, total, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.Notification{}).
		Where("id = ? AND is_read = false", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to mark notification as read", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("notification")
	}
	return nil
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entity.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error; err != nil {
		return domainerrors.Wrap(500, "failed to mark all notifications as read", err)
	}
	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.Notification{}, "id = ?", id)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete notification", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("notification")
	}
	return nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error; err != nil {
		return 0, domainerrors.Wrap(500, "failed to count unread notifications", err)
	}
	return count, nil
}
