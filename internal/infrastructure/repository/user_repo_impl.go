package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"file-management-service/internal/domain/entity"
	domainerrors "file-management-service/internal/domain/errors"
	domrepo "file-management-service/internal/domain/repository"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domrepo.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create user", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("user")
		}
		return nil, domainerrors.Wrap(500, "failed to get user by id", err)
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("user")
		}
		return nil, domainerrors.Wrap(500, "failed to get user by email", err)
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("username = ? AND deleted_at IS NULL", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("user")
		}
		return nil, domainerrors.Wrap(500, "failed to get user by username", err)
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	result := r.db.WithContext(ctx).
		Model(user).
		Where("deleted_at IS NULL").
		Save(user)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update user", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("user")
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to delete user", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("user")
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, filter domrepo.UserFilter) ([]*entity.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.User{}).Where("deleted_at IS NULL")

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("email ILIKE ? OR username ILIKE ? OR full_name ILIKE ?", pattern, pattern, pattern)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to count users", err)
	}

	query = applyOrder(query, filter.OrderBy, filter.OrderDir, "created_at DESC")

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var users []*entity.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, domainerrors.Wrap(500, "failed to list users", err)
	}
	return users, total, nil
}

func (r *userRepository) UpdateStorageUsed(ctx context.Context, userID uuid.UUID, delta int64) error {
	result := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("storage_used", gorm.Expr("storage_used + ?", delta))
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update storage used", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("user")
	}
	return nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID, ip string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"last_login_ip": ip,
		})
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to update last login", result.Error)
	}
	return nil
}

func (r *userRepository) CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return domainerrors.Wrap(500, "failed to create refresh token", err)
	}
	return nil
}

func (r *userRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var token entity.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked = false AND expires_at > ?", tokenHash, time.Now()).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.NotFound("refresh token")
		}
		return nil, domainerrors.Wrap(500, "failed to get refresh token", err)
	}
	return &token, nil
}

func (r *userRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	result := r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked", true)
	if result.Error != nil {
		return domainerrors.Wrap(500, "failed to revoke refresh token", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.NotFound("refresh token")
	}
	return nil
}

func (r *userRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error; err != nil {
		return domainerrors.Wrap(500, "failed to revoke all user tokens", err)
	}
	return nil
}

func (r *userRepository) DeleteExpiredTokens(ctx context.Context) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entity.RefreshToken{}).Error; err != nil {
		return domainerrors.Wrap(500, "failed to delete expired tokens", err)
	}
	return nil
}

func applyOrder(query *gorm.DB, orderBy, orderDir, defaultOrder string) *gorm.DB {
	if orderBy == "" {
		return query.Order(defaultOrder)
	}
	dir := "ASC"
	if orderDir == "desc" {
		dir = "DESC"
	}
	return query.Order(fmt.Sprintf("%s %s", orderBy, dir))
}
