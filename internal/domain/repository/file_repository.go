package repository

import (
	"context"
	"time"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type FileRepository interface {
	Create(ctx context.Context, file *entity.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error)
	GetByStorageKey(ctx context.Context, storageKey string) (*entity.File, error)
	Update(ctx context.Context, file *entity.File) error
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter FileFilter) ([]*entity.File, int64, error)
	CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
	GetByFolder(ctx context.Context, folderID uuid.UUID, filter FileFilter) ([]*entity.File, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.FileStatus) error
	IncrementDownloadCount(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, query string, filter FileFilter) ([]*entity.File, int64, error)

	// File versioning
	CreateVersion(ctx context.Context, version *entity.FileVersion) error
	GetVersions(ctx context.Context, fileID uuid.UUID) ([]*entity.FileVersion, error)

	// Chunked upload tracking
	CreateChunk(ctx context.Context, chunk *entity.FileChunk) error
	GetChunks(ctx context.Context, uploadID string) ([]*entity.FileChunk, error)
	GetChunksByUploadID(ctx context.Context, uploadID string) ([]*entity.FileChunk, error)
	DeleteChunks(ctx context.Context, uploadID string) error
}

type FileFilter struct {
	OwnerID      *uuid.UUID
	FolderID     *uuid.UUID
	MimeType     string
	Extension    string
	Status       *entity.FileStatus
	Tags         []string
	Search       string
	IsPublic     *bool
	MinSize      *int64
	MaxSize      *int64
	CreatedAfter *time.Time
	CreatedBefore *time.Time
	Page         int
	PageSize     int
	OrderBy      string
	OrderDir     string
}
