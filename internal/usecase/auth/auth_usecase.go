package auth

import "context"

// UseCase is the auth business-logic contract.
type UseCase interface {
	// Register creates a new user account and returns a token pair.
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)

	// Login authenticates a user by email/password and issues a token pair.
	Login(ctx context.Context, req *LoginRequest, ip, userAgent string) (*AuthResponse, error)

	// RefreshToken exchanges a valid refresh token for a new token pair.
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*AuthResponse, error)

	// Logout invalidates the given refresh token.
	Logout(ctx context.Context, refreshToken string) error

	// LogoutAll invalidates all active sessions for a user.
	LogoutAll(ctx context.Context, userID string) error

	// ChangePassword verifies the old password and sets a new one,
	// then invalidates all existing sessions.
	ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error

	// GetProfile returns the public profile of a user.
	GetProfile(ctx context.Context, userID string) (*UserResponse, error)

	// UpdateProfile applies partial updates to the user's profile.
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*UserResponse, error)
}
