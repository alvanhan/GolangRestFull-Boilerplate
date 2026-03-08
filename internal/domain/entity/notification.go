package entity

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotifFileShared     NotificationType = "file.shared"
	NotifFileUploaded   NotificationType = "file.uploaded"
	NotifFileDeleted    NotificationType = "file.deleted"
	NotifPermGranted    NotificationType = "permission.granted"
	NotifPermRevoked    NotificationType = "permission.revoked"
	NotifStorageWarning NotificationType = "storage.warning"
	NotifSystemAlert    NotificationType = "system.alert"
)

type Notification struct {
	ID           uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID        `gorm:"type:uuid;not null;index"`
	Type         NotificationType `gorm:"type:varchar(50);not null"`
	Title        string           `gorm:"not null"`
	Message      string           `gorm:"not null"`
	ResourceID   *uuid.UUID       `gorm:"type:uuid"`
	ResourceType *string          `gorm:"type:varchar(20)"`
	IsRead       bool             `gorm:"default:false"`
	ReadAt       *time.Time
	Metadata     JSONMap   `gorm:"type:jsonb"`
	CreatedAt    time.Time `gorm:"index"`

	User User `gorm:"foreignKey:UserID"`
}

// NotificationEvent is used for Redis pub/sub (not stored in DB)
type NotificationEvent struct {
	Type         NotificationType       `json:"type"`
	UserID       string                 `json:"user_id"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}
