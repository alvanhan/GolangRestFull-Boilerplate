package repository

import (
	"context"

	"file-management-service/internal/domain/entity"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByUsername(ctx context.Context, username string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) ([]*entity.User, int64, error)
	UpdateStorageUsed(ctx context.Context, userID uuid.UUID, delta int64) error
	UpdateLastLogin(ctx context.Context, userID uuid.UUID, ip string) error

	// Refresh token management
	CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredTokens(ctx context.Context) error
}

type UserFilter struct {
	Search   string
	Role     *entity.UserRole
	Status   *entity.UserStatus
	Page     int
	PageSize int
	OrderBy  string
	OrderDir string
}
