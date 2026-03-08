package admin

import (
	"context"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/crypto"
	"file-management-service/pkg/utils"

	"github.com/google/uuid"
)

type useCaseImpl struct {
	userRepo repository.UserRepository
	fileRepo repository.FileRepository
}

// NewUseCase constructs the admin UseCase implementation.
func NewUseCase(
	userRepo repository.UserRepository,
	fileRepo repository.FileRepository,
) UseCase {
	return &useCaseImpl{
		userRepo: userRepo,
		fileRepo: fileRepo,
	}
}

func (uc *useCaseImpl) ListUsers(
	ctx context.Context,
	filter *UserListFilter,
) ([]*AdminUserResponse, int64, error) {
	repoFilter := repository.UserFilter{
		Search:   filter.Search,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if filter.Role != "" {
		role := entity.UserRole(filter.Role)
		repoFilter.Role = &role
	}
	if filter.Status != "" {
		status := entity.UserStatus(filter.Status)
		repoFilter.Status = &status
	}
	if repoFilter.Page <= 0 {
		repoFilter.Page = 1
	}
	if repoFilter.PageSize <= 0 {
		repoFilter.PageSize = 20
	}

	users, total, err := uc.userRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, errors.InternalServer(err)
	}

	resp := make([]*AdminUserResponse, len(users))
	for i, u := range users {
		resp[i] = toAdminUserResponse(u)
	}
	return resp, total, nil
}

func (uc *useCaseImpl) CreateUser(
	ctx context.Context,
	req *CreateUserRequest,
) (*AdminUserResponse, error) {
	if existing, _ := uc.userRepo.GetByEmail(ctx, req.Email); existing != nil {
		return nil, errors.Conflict("email already in use")
	}
	if existing, _ := uc.userRepo.GetByUsername(ctx, req.Username); existing != nil {
		return nil, errors.Conflict("username already taken")
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	user := &entity.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		FullName:     req.FullName,
		PasswordHash: hash,
		Role:         entity.UserRole(req.Role),
		Status:       entity.StatusActive,
		StorageQuota: 10 * 1024 * 1024 * 1024, // 10 GB default
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toAdminUserResponse(user), nil
}

func (uc *useCaseImpl) GetUser(ctx context.Context, userID string) (*AdminUserResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}
	return toAdminUserResponse(user), nil
}

func (uc *useCaseImpl) UpdateUser(
	ctx context.Context,
	userID string,
	req *UpdateUserRequest,
) (*AdminUserResponse, error) {
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
	if req.Role != nil {
		user.Role = entity.UserRole(*req.Role)
	}
	if req.Status != nil {
		user.Status = entity.UserStatus(*req.Status)
	}
	if req.StorageQuota != nil {
		user.StorageQuota = *req.StorageQuota
	}
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toAdminUserResponse(user), nil
}

func (uc *useCaseImpl) DeleteUser(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	if _, err := uc.userRepo.GetByID(ctx, uid); err != nil {
		return errors.NotFound("user")
	}
	if err := uc.userRepo.Delete(ctx, uid); err != nil {
		return errors.InternalServer(err)
	}
	return nil
}

func (uc *useCaseImpl) BanUser(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("invalid user ID")
	}
	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return errors.NotFound("user")
	}
	user.Status = entity.StatusBanned
	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return errors.InternalServer(err)
	}
	return nil
}

func (uc *useCaseImpl) GetStats(ctx context.Context) (*StatsResponse, error) {
	_, totalUsers, err := uc.userRepo.List(ctx, repository.UserFilter{Page: 1, PageSize: 1})
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	_, totalFiles, err := uc.fileRepo.List(ctx, repository.FileFilter{Page: 1, PageSize: 1})
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	// Sum storage used across all users
	allUsers, _, err := uc.userRepo.List(ctx, repository.UserFilter{Page: 1, PageSize: 100000})
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	var totalStorage int64
	for _, u := range allUsers {
		totalStorage += u.StorageUsed
	}

	return &StatsResponse{
		TotalUsers:            totalUsers,
		TotalFiles:            totalFiles,
		TotalStorageUsed:      totalStorage,
		TotalStorageFormatted: utils.FormatFileSize(totalStorage),
	}, nil
}

func toAdminUserResponse(u *entity.User) *AdminUserResponse {
	return &AdminUserResponse{
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
