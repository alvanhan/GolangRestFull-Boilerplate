package notification

import (
	"context"
	"fmt"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/logger"
	"file-management-service/pkg/pagination"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type useCaseImpl struct {
	notifRepo repository.NotificationRepository
}

// NewUseCase constructs the notification UseCase implementation.
func NewUseCase(notifRepo repository.NotificationRepository) UseCase {
	return &useCaseImpl{notifRepo: notifRepo}
}

// List returns paginated notifications for a user.
func (uc *useCaseImpl) List(
	ctx context.Context,
	userID string,
	page, pageSize int,
) (*ListNotificationsResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	p := &pagination.Pagination{Page: page, PageSize: pageSize}
	p.Normalize()

	notifs, total, err := uc.notifRepo.GetByUser(ctx, uid, p.Page, p.PageSize)
	if err != nil {
		logger.Error("failed to list notifications", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	resp := make([]*NotificationResponse, len(notifs))
	for i, n := range notifs {
		resp[i] = toNotificationResponse(n)
	}

	return &ListNotificationsResponse{
		Notifications: resp,
		Meta:          p,
		Total:         total,
	}, nil
}

// MarkAsRead marks a single notification as read for the given user.
func (uc *useCaseImpl) MarkAsRead(
	ctx context.Context,
	userID, notificationID string,
) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return errors.BadRequest("invalid notification ID")
	}

	// Verify ownership.
	notif, err := uc.notifRepo.GetByID(ctx, nid)
	if err != nil || notif == nil {
		return errors.NotFound("notification")
	}
	if notif.UserID != uid {
		return errors.Forbidden("access notification")
	}

	if err := uc.notifRepo.MarkAsRead(ctx, nid); err != nil {
		return errors.InternalServer(err)
	}
	return nil
}

// MarkAllAsRead marks every unread notification for the user as read.
func (uc *useCaseImpl) MarkAllAsRead(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	if err := uc.notifRepo.MarkAllAsRead(ctx, uid); err != nil {
		return errors.InternalServer(err)
	}
	return nil
}

// Delete permanently removes a notification that belongs to the user.
func (uc *useCaseImpl) Delete(
	ctx context.Context,
	userID, notificationID string,
) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return errors.BadRequest("invalid notification ID")
	}

	notif, err := uc.notifRepo.GetByID(ctx, nid)
	if err != nil || notif == nil {
		return errors.NotFound("notification")
	}
	if notif.UserID != uid {
		return errors.Forbidden("delete notification")
	}

	if err := uc.notifRepo.Delete(ctx, nid); err != nil {
		return errors.InternalServer(err)
	}
	return nil
}

// GetUnreadCount returns the count of unread notifications for the user.
func (uc *useCaseImpl) GetUnreadCount(
	ctx context.Context,
	userID string,
) (*UnreadCountResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	count, err := uc.notifRepo.GetUnreadCount(ctx, uid)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	return &UnreadCountResponse{Count: count}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────────────────────────────────────

func toNotificationResponse(n *entity.Notification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:       n.ID.String(),
		UserID:   n.UserID.String(),
		Type:     string(n.Type),
		Title:    n.Title,
		Message:  n.Message,
		IsRead:   n.IsRead,
		Metadata: n.Metadata,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
	if n.ResourceID != nil {
		s := n.ResourceID.String()
		resp.ResourceID = &s
	}
	if n.ResourceType != nil {
		resp.ResourceType = n.ResourceType
	}
	if n.ReadAt != nil {
		formatted := fmt.Sprintf("%s", n.ReadAt.Format(time.RFC3339))
		resp.ReadAt = &formatted
	}
	return resp
}
