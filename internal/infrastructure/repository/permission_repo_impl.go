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

type permissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) domrepo.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, permission *entity.Permission) error {
	if err := r.db.WithContext(ctx).Create(permission).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create permission", err)
	}
	return nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	var perm entity.Permission
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&perm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("permission")
		}
		return nil, domainerrors.Wrap(500, "failed to get permission by id", err)
	}
	return &perm, nil
}

func (r *permissionRepository) Update(ctx context.Context, permission *entity.Permission) error {
	result := r.db.WithContext(ctx).Model(permission).Save(permission)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update permission", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("permission")
	}
	return nil
}

func (r *permissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.Permission{}, "id = ?", id)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete permission", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("permission")
	}
	return nil
}

func (r *permissionRepository) GetByResource(ctx context.Context, resourceID uuid.UUID, resourceType entity.ResourceType) ([]*entity.Permission, error) {
	var perms []*entity.Permission
	if err := r.db.WithContext(ctx).
		Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).
		Find(&perms).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get permissions by resource", err)
	}
	return perms, nil
}

func (r *permissionRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Permission, error) {
	var perms []*entity.Permission
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&perms).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get permissions by user", err)
	}
	return perms, nil
}

func (r *permissionRepository) GetByUserAndResource(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, resourceType entity.ResourceType) ([]*entity.Permission, error) {
	var perms []*entity.Permission
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND resource_id = ? AND resource_type = ?", userID, resourceID, resourceType).
		Find(&perms).Error; err != nil {
		return nil, domainerrors.Wrap(500, "failed to get permissions by user and resource", err)
	}
	return perms, nil
}

// HasPermission checks whether a non-expired permission record exists for the given combination.
func (r *permissionRepository) HasPermission(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, resourceType entity.ResourceType, action entity.PermissionAction) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Permission{}).
		Where(
			"user_id = ? AND resource_id = ? AND resource_type = ? AND action = ? AND (expires_at IS NULL OR expires_at > ?)",
			userID, resourceID, resourceType, action, time.Now(),
		).
		Count(&count).Error
	if err != nil {
		return false, domainerrors.Wrap(500, "failed to check permission", err)
	}
	return count > 0, nil
}

func (r *permissionRepository) GrantBulk(ctx context.Context, permissions []*entity.Permission) error {
	if len(permissions) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(permissions).Error; err != nil {
		return domainerrors.Wrap(500, "failed to bulk-grant permissions", err)
	}
	return nil
}

func (r *permissionRepository) RevokeBulk(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).
		Delete(&entity.Permission{}, "id IN ?", ids).Error; err != nil {
		return domainerrors.Wrap(500, "failed to bulk-revoke permissions", err)
	}
	return nil
}

// ─── Share Link ──────────────────────────────────────────────────────────────

func (r *permissionRepository) CreateShareLink(ctx context.Context, link *entity.ShareLink) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create share link", err)
	}
	return nil
}

func (r *permissionRepository) GetShareLink(ctx context.Context, id uuid.UUID) (*entity.ShareLink, error) {
	var link entity.ShareLink
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("share link")
		}
		return nil, domainerrors.Wrap(500, "failed to get share link by id", err)
	}
	return &link, nil
}

func (r *permissionRepository) GetShareLinkByToken(ctx context.Context, token string) (*entity.ShareLink, error) {
	var link entity.ShareLink
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("share link")
		}
		return nil, domainerrors.Wrap(500, "failed to get share link by token", err)
	}
	return &link, nil
}

func (r *permissionRepository) UpdateShareLink(ctx context.Context, link *entity.ShareLink) error {
	result := r.db.WithContext(ctx).Model(link).Save(link)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update share link", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("share link")
	}
	return nil
}

func (r *permissionRepository) DeleteShareLink(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entity.ShareLink{}, "id = ?", id)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete share link", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("share link")
	}
	return nil
}
