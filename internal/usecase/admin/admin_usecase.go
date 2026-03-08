package admin

import (
	"context"
	"time"
)

// CreateUserRequest holds data for creating a new user via the admin interface.
type CreateUserRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	Username string `json:"username"  validate:"required,min=3,max=30,username"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password"  validate:"required,strong_password"`
	Role     string `json:"role"      validate:"required,oneof=super_admin admin manager editor viewer"`
}

// UpdateUserRequest holds optional fields for updating a user via admin.
type UpdateUserRequest struct {
	FullName     *string `json:"full_name"     validate:"omitempty,min=2,max=100"`
	Role         *string `json:"role"          validate:"omitempty,oneof=super_admin admin manager editor viewer"`
	Status       *string `json:"status"        validate:"omitempty,oneof=active inactive banned"`
	StorageQuota *int64  `json:"storage_quota" validate:"omitempty,min=0"`
}

// UserListFilter carries query parameters for listing users.
type UserListFilter struct {
	Search   string `query:"search"`
	Role     string `query:"role"`
	Status   string `query:"status"`
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
}

// AdminUserResponse is the admin view of a user.
type AdminUserResponse struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	FullName      string     `json:"full_name"`
	Role          string     `json:"role"`
	Status        string     `json:"status"`
	Avatar        *string    `json:"avatar,omitempty"`
	StorageQuota  int64      `json:"storage_quota"`
	StorageUsed   int64      `json:"storage_used"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
}

// StatsResponse contains system-wide statistics.
type StatsResponse struct {
	TotalUsers            int64  `json:"total_users"`
	TotalFiles            int64  `json:"total_files"`
	TotalStorageUsed      int64  `json:"total_storage_used"`
	TotalStorageFormatted string `json:"total_storage_formatted"`
}

// UseCase is the admin business-logic contract.
type UseCase interface {
	// ListUsers returns a paginated list of users matching the filter.
	ListUsers(ctx context.Context, filter *UserListFilter) ([]*AdminUserResponse, int64, error)

	// CreateUser creates a new user account with an admin-set role.
	CreateUser(ctx context.Context, req *CreateUserRequest) (*AdminUserResponse, error)

	// GetUser returns details of a specific user.
	GetUser(ctx context.Context, userID string) (*AdminUserResponse, error)

	// UpdateUser applies partial updates to a user record.
	UpdateUser(ctx context.Context, userID string, req *UpdateUserRequest) (*AdminUserResponse, error)

	// DeleteUser permanently removes a user account.
	DeleteUser(ctx context.Context, userID string) error

	// BanUser sets the user's status to banned.
	BanUser(ctx context.Context, userID string) error

	// GetStats returns aggregate system statistics.
	GetStats(ctx context.Context) (*StatsResponse, error)
}
