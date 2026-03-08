package repository

import (
	"context"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type PermissionRepository interface {
	Create(ctx context.Context, permission *entity.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error)
	Update(ctx context.Context, permission *entity.Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByResource(ctx context.Context, resourceID uuid.UUID, resourceType entity.ResourceType) ([]*entity.Permission, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Permission, error)
	GetByUserAndResource(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, resourceType entity.ResourceType) ([]*entity.Permission, error)
	HasPermission(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, resourceType entity.ResourceType, action entity.PermissionAction) (bool, error)
	GrantBulk(ctx context.Context, permissions []*entity.Permission) error
	RevokeBulk(ctx context.Context, ids []uuid.UUID) error

	CreateShareLink(ctx context.Context, link *entity.ShareLink) error
	GetShareLink(ctx context.Context, id uuid.UUID) (*entity.ShareLink, error)
	GetShareLinkByToken(ctx context.Context, token string) (*entity.ShareLink, error)
	UpdateShareLink(ctx context.Context, link *entity.ShareLink) error
	DeleteShareLink(ctx context.Context, id uuid.UUID) error
}
