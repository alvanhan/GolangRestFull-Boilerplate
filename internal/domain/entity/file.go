package entity

import (
	"time"

	"github.com/google/uuid"
)

type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"
	FileStatusProcessing FileStatus = "processing"
	FileStatusReady      FileStatus = "ready"
	FileStatusError      FileStatus = "error"
	FileStatusDeleted    FileStatus = "deleted"
)

// StringArray for PostgreSQL text[] type
type StringArray []string

type File struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name           string     `gorm:"not null"`
	OriginalName   string     `gorm:"not null"`
	Extension      string     `gorm:"not null"`
	MimeType       string     `gorm:"not null"`
	Size           int64      `gorm:"not null"`
	Checksum       string     `gorm:"not null"` // SHA256
	StorageKey     string     `gorm:"not null;uniqueIndex"` // MinIO object key
	StorageBucket  string     `gorm:"not null"`
	FolderID       *uuid.UUID `gorm:"type:uuid;index"` // nil = root
	OwnerID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	Version        int        `gorm:"default:1"`
	Status         FileStatus `gorm:"type:varchar(20);default:'pending'"`
	IsEncrypted    bool       `gorm:"default:false"`
	IsPublic       bool       `gorm:"default:false"`
	DownloadCount  int64      `gorm:"default:0"`
	LastAccessedAt *time.Time
	ExpiresAt      *time.Time // for temporary shared files
	Tags           StringArray `gorm:"type:text[]"`
	Metadata       JSONMap    `gorm:"type:jsonb"`
	ThumbnailKey   *string    // MinIO key for thumbnail
	Description    *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"index"`

	Owner       User          `gorm:"foreignKey:OwnerID"`
	Folder      *Folder       `gorm:"foreignKey:FolderID"`
	Versions    []FileVersion `gorm:"foreignKey:FileID"`
}

type FileVersion struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FileID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Version     int       `gorm:"not null"`
	StorageKey  string    `gorm:"not null"`
	Size        int64     `gorm:"not null"`
	Checksum    string    `gorm:"not null"`
	ChangedByID uuid.UUID `gorm:"type:uuid;not null"`
	ChangeNote  *string
	CreatedAt   time.Time

	File      File `gorm:"foreignKey:FileID"`
	ChangedBy User `gorm:"foreignKey:ChangedByID"`
}

// FileChunk tracks chunks for a multipart/chunked upload session
type FileChunk struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UploadID    string    `gorm:"not null;index"` // session ID for chunked upload
	FileKey     string    `gorm:"not null"`       // intended final storage key
	ChunkIndex  int       `gorm:"not null"`
	ChunkSize   int64     `gorm:"not null"`
	TotalChunks int       `gorm:"not null"`
	StorageKey  string    `gorm:"not null"` // temp storage key
	Checksum    string    `gorm:"not null"`
	UploadedBy  uuid.UUID `gorm:"type:uuid;not null"`
	ExpiresAt   time.Time `gorm:"not null"`
	CreatedAt   time.Time
}

func (f *File) IsOwnedBy(userID uuid.UUID) bool { return f.OwnerID == userID }
func (f *File) IsReady() bool                   { return f.Status == FileStatusReady }
func (f *File) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}
