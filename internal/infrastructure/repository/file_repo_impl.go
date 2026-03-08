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

type fileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) domrepo.FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) base(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Where("deleted_at IS NULL")
}

func (r *fileRepository) Create(ctx context.Context, file *entity.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create file", err)
	}
	return nil
}

func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	var file entity.File
	err := r.base(ctx).Where("id = ?", id).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("file")
		}
		return nil, domainerrors.Wrap(500, "failed to get file by id", err)
	}
	return &file, nil
}

func (r *fileRepository) GetByStorageKey(ctx context.Context, storageKey string) (*entity.File, error) {
	var file entity.File
	err := r.base(ctx).Where("storage_key = ?", storageKey).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("file")
		}
		return nil, domainerrors.Wrap(500, "failed to get file by storage key", err)
	}
	return &file, nil
}

func (r *fileRepository) Update(ctx context.Context, file *entity.File) error {
	result := r.db.WithContext(ctx).
		Model(file).
		Where("deleted_at IS NULL").
		Save(file)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update file", result.Error)
	}
	return nil
}

// Delete permanently removes the file record from the database.
func (r *fileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Delete(&entity.File{}, "id = ?", id)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete file", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("file")
	}
	return nil
}

// SoftDelete marks the file as deleted by setting deleted_at.
func (r *fileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.File{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to soft delete file", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("file")
	}
	return nil
}

func (r *fileRepository) List(ctx context.Context, filter domrepo.FileFilter) ([]*entity.File, int64, error) {
	query := r.applyFilter(r.base(ctx).Model(&entity.File{}), filter)

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR original_name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count files", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var files []*entity.File
	if err := query.Find(&files).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to list files", err)
	}
	return files, total, nil
}

func (r *fileRepository) CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.base(ctx).Model(&entity.File{}).
		Where("owner_id = ?", ownerID).
		Count(&count).Error; err != nil {
		return 0, domainerrors.Wrap(500, "failed to count files by owner", err)
	}
	return count, nil
}

func (r *fileRepository) GetByFolder(ctx context.Context, folderID uuid.UUID, filter domrepo.FileFilter) ([]*entity.File, int64, error) {
	query := r.applyFilter(
		r.base(ctx).Model(&entity.File{}).Where("folder_id = ?", folderID),
		filter,
	)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count files by folder", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var files []*entity.File
	if err := query.Find(&files).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to get files by folder", err)
	}
	return files, total, nil
}

func (r *fileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.FileStatus) error {
	result := r.db.WithContext(ctx).
		Model(&entity.File{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("status", status)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update file status", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("file")
	}
	return nil
}

func (r *fileRepository) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entity.File{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"download_count":   gorm.Expr("download_count + 1"),
			"last_accessed_at": now,
		}).Error; err != nil {
		return domainerrors.Wrap(500, "failed to increment download count", err)
	}
	return nil
}

// Search finds files whose name, original name, or description matches query (case-insensitive).
func (r *fileRepository) Search(ctx context.Context, query string, filter domrepo.FileFilter) ([]*entity.File, int64, error) {
	pattern := "%" + query + "%"
	q := r.applyFilter(
		r.base(ctx).Model(&entity.File{}).
			Where("name ILIKE ? OR original_name ILIKE ? OR description ILIKE ?", pattern, pattern, pattern),
		filter,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count search results", err)
	}

	q = applyOrder(q, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var files []*entity.File
	if err := q.Find(&files).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to search files", err)
	}
	return files, total, nil
}

func (r *fileRepository) CreateVersion(ctx context.Context, version *entity.FileVersion) error {
	if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create file version", err)
	}
	return nil
}

func (r *fileRepository) GetVersions(ctx context.Context, fileID uuid.UUID) ([]*entity.FileVersion, error) {
	var versions []*entity.FileVersion
	if err := r.db.WithContext(ctx).
		Where("file_id = ?", fileID).
		Order("version DESC").
		Find(&versions).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get file versions", err)
	}
	return versions, nil
}

func (r *fileRepository) CreateChunk(ctx context.Context, chunk *entity.FileChunk) error {
	if err := r.db.WithContext(ctx).Create(chunk).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create file chunk", err)
	}
	return nil
}

func (r *fileRepository) GetChunks(ctx context.Context, uploadID string) ([]*entity.FileChunk, error) {
	var chunks []*entity.FileChunk
	if err := r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get file chunks", err)
	}
	return chunks, nil
}

func (r *fileRepository) GetChunksByUploadID(ctx context.Context, uploadID string) ([]*entity.FileChunk, error) {
	return r.GetChunks(ctx, uploadID)
}

func (r *fileRepository) DeleteChunks(ctx context.Context, uploadID string) error {
	if err := r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Delete(&entity.FileChunk{}).Error; err != nil {
		return domainerrors.Wrap(500, "failed to delete file chunks", err)
	}
	return nil
}

// applyFilter applies all non-zero FileFilter fields to the query.
func (r *fileRepository) applyFilter(query *gorm.DB, f domrepo.FileFilter) *gorm.DB {
	if f.OwnerID != nil {
		query = query.Where("owner_id = ?", *f.OwnerID)
	}
	if f.FolderID != nil {
		query = query.Where("folder_id = ?", *f.FolderID)
	}
	if f.MimeType != "" {
		query = query.Where("mime_type = ?", f.MimeType)
	}
	if f.Extension != "" {
		query = query.Where("extension = ?", f.Extension)
	}
	if f.Status != nil {
		query = query.Where("status = ?", *f.Status)
	}
	// Use = ANY(tags) so each required tag must appear in the file's tag array.
	for _, tag := range f.Tags {
		query = query.Where("? = ANY(tags)", tag)
	}
	if f.IsPublic != nil {
		query = query.Where("is_public = ?", *f.IsPublic)
	}
	if f.MinSize != nil {
		query = query.Where("size >= ?", *f.MinSize)
	}
	if f.MaxSize != nil {
		query = query.Where("size <= ?", *f.MaxSize)
	}
	if f.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *f.CreatedAfter)
	}
	if f.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *f.CreatedBefore)
	}
	return query
}
