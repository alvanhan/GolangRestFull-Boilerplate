package auth

import (
	"context"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/crypto"
	pkgjwt "file-management-service/pkg/jwt"
	"file-management-service/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type useCaseImpl struct {
	userRepo   repository.UserRepository
	jwtService pkgjwt.JWTService
}

func NewUseCase(userRepo repository.UserRepository, jwtService pkgjwt.JWTService) UseCase {
	return &useCaseImpl{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (uc *useCaseImpl) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	if existing, _ := uc.userRepo.GetByEmail(ctx, req.Email); existing != nil {
		return nil, errors.Conflict("email already in use")
	}
	if existing, _ := uc.userRepo.GetByUsername(ctx, req.Username); existing != nil {
		return nil, errors.Conflict("username already taken")
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	user := &entity.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		FullName:     req.FullName,
		PasswordHash: hash,
		Role:         entity.RoleViewer,
		Status:       entity.StatusActive,
		StorageQuota: 10 * 1024 * 1024 * 1024, // 10 GB
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		logger.Error("failed to create user", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	return uc.buildAuthResponse(ctx, user)
}

func (uc *useCaseImpl) Login(ctx context.Context, req *LoginRequest, ip, userAgent string) (*AuthResponse, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, errors.Unauthorized("invalid credentials")
	}

	if !user.IsActive() {
		return nil, errors.Forbidden("account is not active")
	}

	if err := crypto.CheckPassword(req.Password, user.PasswordHash); err != nil {
		return nil, errors.Unauthorized("invalid credentials")
	}

	if err := uc.userRepo.UpdateLastLogin(ctx, user.ID, ip); err != nil {
		logger.Warn("failed to update last login", zap.Error(err), zap.String("user_id", user.ID.String()))
	}

	resp, err := uc.buildAuthResponse(ctx, user)
	if err != nil {
		return nil, err
	}

	// Persist the refresh token so it can be revoked later.
	tokenHash := crypto.HashSHA256(resp.RefreshToken)
	rt := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		IPAddress: ip,
		UserAgent: userAgent,
		ExpiresAt: resp.ExpiresAt.Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := uc.userRepo.CreateRefreshToken(ctx, rt); err != nil {
		logger.Error("failed to store refresh token", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	return resp, nil
}

func (uc *useCaseImpl) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*AuthResponse, error) {
	claims, err := uc.jwtService.ValidateToken(req.RefreshToken, pkgjwt.RefreshToken)
	if err != nil {
		return nil, errors.Unauthorized("invalid or expired refresh token")
	}

	tokenHash := crypto.HashSHA256(req.RefreshToken)
	stored, err := uc.userRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil || stored == nil || stored.Revoked {
		return nil, errors.Unauthorized("refresh token has been revoked")
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, errors.Unauthorized("refresh token expired")
	}

	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, errors.Unauthorized("user not found")
	}
	if !user.IsActive() {
		return nil, errors.Forbidden("account is not active")
	}

	// Rotate: revoke old token.
	if err := uc.userRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		logger.Warn("failed to revoke old refresh token", zap.Error(err))
	}

	resp, err := uc.buildAuthResponse(ctx, user)
	if err != nil {
		return nil, err
	}

	newHash := crypto.HashSHA256(resp.RefreshToken)
	newRT := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newHash,
		IPAddress: stored.IPAddress,
		UserAgent: stored.UserAgent,
		ExpiresAt: stored.ExpiresAt,
		CreatedAt: time.Now(),
	}
	if err := uc.userRepo.CreateRefreshToken(ctx, newRT); err != nil {
		logger.Error("failed to store rotated refresh token", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	return resp, nil
}

func (uc *useCaseImpl) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := crypto.HashSHA256(refreshToken)
	if err := uc.userRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		logger.Warn("logout: failed to revoke token", zap.Error(err))
	}
	return nil
}

func (uc *useCaseImpl) LogoutAll(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	if err := uc.userRepo.RevokeAllUserTokens(ctx, uid); err != nil {
		logger.Error("failed to revoke all user tokens", zap.Error(err), zap.String("user_id", userID))
		return errors.InternalServer(err)
	}
	return nil
}

func (uc *useCaseImpl) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}

	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return errors.NotFound("user")
	}

	if err := crypto.CheckPassword(req.OldPassword, user.PasswordHash); err != nil {
		return errors.Unauthorized("current password is incorrect")
	}

	newHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return errors.InternalServer(err)
	}

	user.PasswordHash = newHash
	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		logger.Error("failed to update password", zap.Error(err))
		return errors.InternalServer(err)
	}

	// Invalidate all existing sessions.
	if err := uc.userRepo.RevokeAllUserTokens(ctx, uid); err != nil {
		logger.Warn("failed to revoke tokens after password change", zap.Error(err))
	}

	return nil
}

func (uc *useCaseImpl) GetProfile(ctx context.Context, userID string) (*UserResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}
	return toUserResponse(user), nil
}

func (uc *useCaseImpl) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*UserResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Avatar != nil {
		user.Avatar = req.Avatar
	}
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		logger.Error("failed to update profile", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	return toUserResponse(user), nil
}

func (uc *useCaseImpl) buildAuthResponse(ctx context.Context, user *entity.User) (*AuthResponse, error) {
	pair, err := uc.jwtService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		logger.Error("failed to generate token pair", zap.Error(err))
		return nil, errors.InternalServer(err)
	}
	return &AuthResponse{
		User:         *toUserResponse(user),
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
	}, nil
}

func toUserResponse(u *entity.User) *UserResponse {
	return &UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Username:      u.Username,
		FullName:      u.FullName,
		Role:          string(u.Role),
		Status:        string(u.Status),
		Avatar:        u.Avatar,
		StorageQuota:  u.StorageQuota,
		StorageUsed:   u.StorageUsed,
		LastLoginAt:   u.LastLoginAt,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}
