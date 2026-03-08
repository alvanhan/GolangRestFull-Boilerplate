package handler

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/delivery/http/middleware"
	"file-management-service/internal/usecase/auth"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

type AuthHandler struct {
	authUC    auth.UseCase
	validator *validator.Validator
}

func NewAuthHandler(authUC auth.UseCase, v *validator.Validator) *AuthHandler {
	return &AuthHandler{authUC: authUC, validator: v}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth.RegisterRequest  true  "Registration payload"
// @Success      201  {object}  response.Response{data=auth.AuthResponse}
// @Failure      400  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req auth.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	result, err := h.authUC.Register(c.Context(), &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "registration successful", result)
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user and get JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth.LoginRequest  true  "Login credentials"
// @Success      200  {object}  response.Response{data=auth.AuthResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	result, err := h.authUC.Login(c.Context(), &req, c.IP(), c.Get(fiber.HeaderUserAgent))
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "login successful", result)
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Get a new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth.RefreshTokenRequest  true  "Refresh token"
// @Success      200  {object}  response.Response{data=auth.AuthResponse}
// @Failure      401  {object}  response.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req auth.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	result, err := h.authUC.RefreshToken(c.Context(), &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "token refreshed", result)
}

// Logout godoc
// @Summary      Logout
// @Description  Invalidate refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      auth.RefreshTokenRequest  true  "Refresh token"
// @Success      204
// @Failure      401  {object}  response.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req auth.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if err := h.authUC.Logout(c.Context(), req.RefreshToken); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// LogoutAll godoc
// @Summary      Logout all sessions
// @Description  Invalidate all refresh tokens for the current user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      204
// @Failure      401  {object}  response.Response
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if err := h.authUC.LogoutAll(c.Context(), userID); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the authenticated user's password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      auth.ChangePasswordRequest  true  "Password change payload"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req auth.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	if err := h.authUC.ChangePassword(c.Context(), userID, &req); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "password changed successfully", nil)
}

// GetProfile godoc
// @Summary      Get current user profile
// @Description  Get the authenticated user's profile information
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=auth.UserResponse}
// @Failure      401  {object}  response.Response
// @Router       /auth/me [get]
func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	result, err := h.authUC.GetProfile(c.Context(), userID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "profile retrieved", result)
}

// UpdateProfile godoc
// @Summary      Update current user profile
// @Description  Update the authenticated user's profile information
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      auth.UpdateProfileRequest  true  "Profile update payload"
// @Success      200  {object}  response.Response{data=auth.UserResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/me [put]
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	var req auth.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	result, err := h.authUC.UpdateProfile(c.Context(), userID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "profile updated", result)
}
