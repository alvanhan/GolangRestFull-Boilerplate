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

type folderRepository struct {
	db *gorm.DB
}

func NewFolderRepository(db *gorm.DB) domrepo.FolderRepository {
	return &folderRepository{db: db}
}

func (r *folderRepository) base(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Where("deleted_at IS NULL")
}

func (r *folderRepository) Create(ctx context.Context, folder *entity.Folder) error {
	if err := r.db.WithContext(ctx).Create(folder).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create folder", err)
	}
	return nil
}

func (r *folderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error) {
	var folder entity.Folder
	err := r.base(ctx).Where("id = ?", id).First(&folder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("folder")
		}
		return nil, domainerrors.Wrap(500, "failed to get folder by id", err)
	}
	return &folder, nil
}

func (r *folderRepository) GetByPath(ctx context.Context, path string, ownerID uuid.UUID) (*entity.Folder, error) {
	var folder entity.Folder
	err := r.base(ctx).
		Where("path = ? AND owner_id = ?", path, ownerID).
		First(&folder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("folder")
		}
		return nil, domainerrors.Wrap(500, "failed to get folder by path", err)
	}
	return &folder, nil
}

func (r *folderRepository) Update(ctx context.Context, folder *entity.Folder) error {
	result := r.db.WithContext(ctx).
		Model(folder).
		Where("deleted_at IS NULL").
		Save(folder)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update folder", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("folder")
	}
	return nil
}

// Delete permanently removes the folder record.
func (r *folderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Delete(&entity.Folder{}, "id = ?", id)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete folder", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("folder")
	}
	return nil
}

// SoftDelete marks the folder as deleted by setting deleted_at.
func (r *folderRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.Folder{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to soft delete folder", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("folder")
	}
	return nil
}

func (r *folderRepository) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*entity.Folder, error) {
	var folders []*entity.Folder
	if err := r.base(ctx).
		Where("parent_id = ?", parentID).
		Order("name ASC").
		Find(&folders).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get child folders", err)
	}
	return folders, nil
}

func (r *folderRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error) {
	var folders []*entity.Folder
	if err := r.base(ctx).
		Where("owner_id = ?", ownerID).
		Order("name ASC").
		Find(&folders).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get folders by owner", err)
	}
	return folders, nil
}

// Move updates the folder's parent and recalculates materialized paths for the
// folder and all its descendants atomically.
func (r *folderRepository) Move(ctx context.Context, folderID uuid.UUID, newParentID *uuid.UUID) error {
	var current entity.Folder
	if err := r.base(ctx).Where("id = ?", folderID).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domainerrors.NotFound("folder")
		}
		return domainerrors.Wrap(500, "failed to get folder for move", err)
	}

	// oldFullPath is used to match descendants whose path starts with this folder.
	oldFullPath := current.Path + "/" + folderID.String()

	var newParentPath string
	if newParentID != nil {
		var parent entity.Folder
		if err := r.base(ctx).Where("id = ?", *newParentID).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.NotFound("parent folder")
			}
			return domainerrors.Wrap(500, "failed to get parent folder", err)
		}
		newParentPath = parent.Path + "/" + parent.ID.String()
	}
	// newFullPath is this folder's new materialized full path.
	newFullPath := newParentPath + "/" + folderID.String()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update the moved folder itself.
		if err := tx.Model(&entity.Folder{}).
			Where("id = ? AND deleted_at IS NULL", folderID).
			Updates(map[string]interface{}{
				"parent_id": newParentID,
				"path":      newParentPath,
			}).Error; err != nil {
			return domainerrors.Wrap(500, "failed to update folder path", err)
		}

		// Rewrite paths of all descendants using PostgreSQL REPLACE().
		if err := tx.Model(&entity.Folder{}).
			Where("path LIKE ? AND deleted_at IS NULL", oldFullPath+"/%").
			Update("path", gorm.Expr("REPLACE(path, ?, ?)", oldFullPath, newFullPath)).
			Error; err != nil {
			return domainerrors.Wrap(500, "failed to update descendant paths", err)
		}

		return nil
	})
}

func (r *folderRepository) UpdateCounts(ctx context.Context, folderID uuid.UUID, fileDelta, folderDelta int64, sizeDelta int64) error {
	result := r.db.WithContext(ctx).
		Model(&entity.Folder{}).
		Where("id = ? AND deleted_at IS NULL", folderID).
		Updates(map[string]interface{}{
			"file_count":   gorm.Expr("file_count + ?", fileDelta),
			"folder_count": gorm.Expr("folder_count + ?", folderDelta),
			"size":         gorm.Expr("size + ?", sizeDelta),
		})
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update folder counts", result.Error)
	}
	return nil
}

func (r *folderRepository) List(ctx context.Context, filter domrepo.FolderFilter) ([]*entity.Folder, int64, error) {
	query := r.applyFilter(r.base(ctx).Model(&entity.Folder{}), filter)

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ?", pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count folders", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "name ASC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var folders []*entity.Folder
	if err := query.Find(&folders).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to list folders", err)
	}
	return folders, total, nil
}

func (r *folderRepository) Search(ctx context.Context, query string, filter domrepo.FolderFilter) ([]*entity.Folder, int64, error) {
	pattern := "%" + query + "%"
	q := r.applyFilter(
		r.base(ctx).Model(&entity.Folder{}).Where("name ILIKE ?", pattern),
		filter,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count folder search results", err)
	}

	q = applyOrder(q, filter.OrderBy, filter.OrderDir, "name ASC")

	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var folders []*entity.Folder
	if err := q.Find(&folders).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to search folders", err)
	}
	return folders, total, nil
}

func (r *folderRepository) applyFilter(query *gorm.DB, f domrepo.FolderFilter) *gorm.DB {
	if f.OwnerID != nil {
		query = query.Where("owner_id = ?", *f.OwnerID)
	}
	if f.ParentID != nil {
		query = query.Where("parent_id = ?", *f.ParentID)
	}
	if f.IsRoot != nil {
		query = query.Where("is_root = ?", *f.IsRoot)
	}
	if f.IsShared != nil {
		query = query.Where("is_shared = ?", *f.IsShared)
	}
	return query
}
