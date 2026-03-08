package entity

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string     `gorm:"not null"`
	Description *string
	OwnerID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	ParentID    *uuid.UUID `gorm:"type:uuid;index"` // nil = root
	Path        string     `gorm:"not null"`        // materialized path e.g. "/uuid1/uuid2/current"
	IsRoot      bool       `gorm:"default:false"`
	IsShared    bool       `gorm:"default:false"`
	Color       *string    // for UI coloring
	Icon        *string
	Size        int64 `gorm:"default:0"` // total size of contents
	FileCount   int64 `gorm:"default:0"`
	FolderCount int64 `gorm:"default:0"`
	Metadata    JSONMap `gorm:"type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`

	Owner       User         `gorm:"foreignKey:OwnerID"`
	Parent      *Folder      `gorm:"foreignKey:ParentID"`
	Children    []Folder     `gorm:"foreignKey:ParentID"`
	Files       []File       `gorm:"foreignKey:FolderID"`
}

func (f *Folder) IsOwnedBy(userID uuid.UUID) bool { return f.OwnerID == userID }
func (f *Folder) GetFullPath() string              { return f.Path + "/" + f.ID.String() }
