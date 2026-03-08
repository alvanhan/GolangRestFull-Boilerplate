package repository

import (
	"context"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type FolderRepository interface {
	Create(ctx context.Context, folder *entity.Folder) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error)
	GetByPath(ctx context.Context, path string, ownerID uuid.UUID) (*entity.Folder, error)
	Update(ctx context.Context, folder *entity.Folder) error
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]*entity.Folder, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error)
	Move(ctx context.Context, folderID uuid.UUID, newParentID *uuid.UUID) error
	UpdateCounts(ctx context.Context, folderID uuid.UUID, fileDelta, folderDelta int64, sizeDelta int64) error
	List(ctx context.Context, filter FolderFilter) ([]*entity.Folder, int64, error)
	Search(ctx context.Context, query string, filter FolderFilter) ([]*entity.Folder, int64, error)
}

type FolderFilter struct {
	OwnerID  *uuid.UUID
	ParentID *uuid.UUID
	IsRoot   *bool
	IsShared *bool
	Search   string
	Page     int
	PageSize int
	OrderBy  string
	OrderDir string
}
