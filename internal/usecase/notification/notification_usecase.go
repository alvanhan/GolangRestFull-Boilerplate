package notification

import (
	"context"

	"file-management-service/pkg/pagination"
)

// NotificationResponse is the public representation of a notification.
type NotificationResponse struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Type         string                 `json:"type"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	ResourceType *string                `json:"resource_type,omitempty"`
	IsRead       bool                   `json:"is_read"`
	ReadAt       *string                `json:"read_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

// ListNotificationsResponse wraps a paginated list of notifications.
type ListNotificationsResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	Meta          *pagination.Pagination  `json:"meta"`
	Total         int64                   `json:"total"`
}

// UnreadCountResponse carries the count of unread notifications.
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// UseCase is the notification business-logic contract.
type UseCase interface {
	// List returns paginated notifications for a user.
	List(ctx context.Context, userID string, page, pageSize int) (*ListNotificationsResponse, error)

	// MarkAsRead marks a single notification as read.
	MarkAsRead(ctx context.Context, userID, notificationID string) error

	// MarkAllAsRead marks every unread notification for the user as read.
	MarkAllAsRead(ctx context.Context, userID string) error

	// Delete permanently removes a notification.
	Delete(ctx context.Context, userID, notificationID string) error

	// GetUnreadCount returns the number of unread notifications for a user.
	GetUnreadCount(ctx context.Context, userID string) (*UnreadCountResponse, error)
}
