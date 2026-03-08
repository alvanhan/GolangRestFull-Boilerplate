package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditFileUpload       AuditAction = "file.upload"
	AuditFileDownload     AuditAction = "file.download"
	AuditFileDelete       AuditAction = "file.delete"
	AuditFileMove         AuditAction = "file.move"
	AuditFileCopy         AuditAction = "file.copy"
	AuditFileRename       AuditAction = "file.rename"
	AuditFileShare        AuditAction = "file.share"
	AuditFileView         AuditAction = "file.view"
	AuditFolderCreate     AuditAction = "folder.create"
	AuditFolderDelete     AuditAction = "folder.delete"
	AuditFolderMove       AuditAction = "folder.move"
	AuditFolderRename     AuditAction = "folder.rename"
	AuditPermissionGrant  AuditAction = "permission.grant"
	AuditPermissionRevoke AuditAction = "permission.revoke"
	AuditUserLogin        AuditAction = "user.login"
	AuditUserLogout       AuditAction = "user.logout"
	AuditUserCreate       AuditAction = "user.create"
	AuditUserUpdate       AuditAction = "user.update"
	AuditUserDelete       AuditAction = "user.delete"
)

type AuditLog struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       *uuid.UUID  `gorm:"type:uuid;index"` // nil for system actions
	Action       AuditAction `gorm:"type:varchar(50);not null;index"`
	ResourceID   *uuid.UUID  `gorm:"type:uuid;index"`
	ResourceType *string     `gorm:"type:varchar(20)"`
	ResourceName *string
	IPAddress    string  `gorm:"not null"`
	UserAgent    string
	Details      JSONMap `gorm:"type:jsonb"`
	OldValues    JSONMap `gorm:"type:jsonb"`
	NewValues    JSONMap `gorm:"type:jsonb"`
	Status       string  `gorm:"type:varchar(10);default:'success'"` // success, failed
	ErrorMessage *string
	Duration     int64     // request duration in ms
	CreatedAt    time.Time `gorm:"index"`

	User *User `gorm:"foreignKey:UserID"`
}
