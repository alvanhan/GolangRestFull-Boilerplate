package auth

import "time"

// RegisterRequest holds the data needed to create a new user account.
type RegisterRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	Username string `json:"username"  validate:"required,min=3,max=30,username"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password"  validate:"required,strong_password"`
}

// LoginRequest holds credentials for a login attempt.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest carries a refresh token to exchange for a new token pair.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest is used to change the authenticated user's password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,strong_password"`
}

// ForgotPasswordRequest initiates a password-reset flow.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// UpdateProfileRequest contains optional fields a user may update on their profile.
type UpdateProfileRequest struct {
	FullName *string `json:"full_name" validate:"omitempty,min=2,max=100"`
	Avatar   *string `json:"avatar"    validate:"omitempty,url"`
}

// AuthResponse is returned after successful authentication operations.
type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// UserResponse is the public representation of a user.
type UserResponse struct {
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
