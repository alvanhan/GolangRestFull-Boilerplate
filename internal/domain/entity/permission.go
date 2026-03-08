package entity

import (
	"time"

	"github.com/google/uuid"
)

type PermissionAction string
type ResourceType string

const (
	ActionRead              PermissionAction = "read"
	ActionWrite             PermissionAction = "write"
	ActionDelete            PermissionAction = "delete"
	ActionShare             PermissionAction = "share"
	ActionDownload          PermissionAction = "download"
	ActionUpload            PermissionAction = "upload"
	ActionManagePermissions PermissionAction = "manage_permissions"
)

const (
	ResourceTypeFile   ResourceType = "file"
	ResourceTypeFolder ResourceType = "folder"
)

// Permission represents a specific access grant on a resource
type Permission struct {
	ID           uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ResourceID   uuid.UUID        `gorm:"type:uuid;not null;index"`
	ResourceType ResourceType     `gorm:"type:varchar(20);not null"`
	UserID       uuid.UUID        `gorm:"type:uuid;not null;index"`
	Action       PermissionAction `gorm:"type:varchar(30);not null"`
	GrantedByID  uuid.UUID        `gorm:"type:uuid;not null"`
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time

	User      User `gorm:"foreignKey:UserID"`
	GrantedBy User `gorm:"foreignKey:GrantedByID"`
}

// ShareLink represents a public or restricted share link
type ShareLink struct {
	ID           uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Token        string           `gorm:"uniqueIndex;not null"`
	ResourceID   uuid.UUID        `gorm:"type:uuid;not null;index"`
	ResourceType ResourceType     `gorm:"type:varchar(20);not null"`
	CreatedByID  uuid.UUID        `gorm:"type:uuid;not null"`
	Action       PermissionAction `gorm:"type:varchar(30);not null;default:'read'"`
	Password     *string          // optional password protection
	ExpiresAt    *time.Time
	MaxUses      *int
	UseCount     int  `gorm:"default:0"`
	IsActive     bool `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	CreatedBy User `gorm:"foreignKey:CreatedByID"`
}

func (p *Permission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

func (sl *ShareLink) IsExpired() bool {
	if sl.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sl.ExpiresAt)
}

func (sl *ShareLink) IsUsable() bool {
	if !sl.IsActive || sl.IsExpired() {
		return false
	}
	if sl.MaxUses != nil && sl.UseCount >= *sl.MaxUses {
		return false
	}
	return true
}
