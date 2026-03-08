package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"file-management-service/internal/domain/entity"
	domrepo "file-management-service/internal/domain/repository"
	"file-management-service/pkg/pagination"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

// AdminHandler handles admin-only user management and stats.
type AdminHandler struct {
	userRepo  domrepo.UserRepository
	fileRepo  domrepo.FileRepository
	validator *validator.Validator
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(userRepo domrepo.UserRepository, fileRepo domrepo.FileRepository, v *validator.Validator) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, fileRepo: fileRepo, validator: v}
}

// ListUsers godoc
// @Summary      List users
// @Description  Get paginated list of all users (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page" default(1)
// @Param        page_size  query  int     false  "Page size" default(20)
// @Param        search     query  string  false  "Search by name or email"
// @Success      200  {object}  response.Response
// @Router       /admin/users [get]
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	pag, err := pagination.ParseFromQuery(c)
	if err != nil {
		return response.BadRequest(c, "invalid pagination")
	}

	filter := domrepo.UserFilter{
		Search:   c.Query("search"),
		Page:     pag.Page,
		PageSize: pag.PageSize,
		OrderBy:  c.Query("order_by", "created_at"),
		OrderDir: c.Query("order_dir", "desc"),
	}
	if role := c.Query("role"); role != "" {
		r := entity.UserRole(role)
		filter.Role = &r
	}
	if status := c.Query("status"); status != "" {
		s := entity.UserStatus(status)
		filter.Status = &s
	}

	users, total, err := h.userRepo.List(c.Context(), filter)
	if err != nil {
		return handleError(c, err)
	}

	meta := pagination.NewMeta(pag.Page, pag.PageSize, total)
	return response.SuccessWithMeta(c, fiber.StatusOK, "users retrieved", users, meta)
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get user details by ID (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid UUID")
	}

	user, err := h.userRepo.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "user retrieved", user)
}

// CreateUser godoc
// @Summary      Create user (admin)
// @Description  Create a new user as admin
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object  true  "User payload"
// @Success      201  {object}  response.Response
// @Router       /admin/users [post]
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req struct {
		Email    string           `json:"email"     validate:"required,email"`
		Username string           `json:"username"  validate:"required,min=3,max=30"`
		FullName string           `json:"full_name" validate:"required"`
		Password string           `json:"password"  validate:"required,strong_password"`
		Role     entity.UserRole  `json:"role"      validate:"required,oneof=super_admin admin manager editor viewer"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	user := &entity.User{
		Email:    req.Email,
		Username: req.Username,
		FullName: req.FullName,
		Role:     req.Role,
		Status:   entity.StatusActive,
	}

	if err := h.userRepo.Create(c.Context(), user); err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "user created", user)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update user information (admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                    true  "User ID (UUID)"
// @Param        request  body  object  true  "Update payload"
// @Success      200  {object}  response.Response
// @Router       /admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid UUID")
	}

	user, err := h.userRepo.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	var req struct {
		FullName     *string          `json:"full_name"`
		Role         *entity.UserRole `json:"role"         validate:"omitempty,oneof=super_admin admin manager editor viewer"`
		Status       *entity.UserStatus `json:"status"     validate:"omitempty,oneof=active inactive banned"`
		StorageQuota *int64           `json:"storage_quota"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.StorageQuota != nil {
		user.StorageQuota = *req.StorageQuota
	}

	if err := h.userRepo.Update(c.Context(), user); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "user updated", user)
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Soft-delete a user (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID (UUID)"
// @Success      204
// @Router       /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid UUID")
	}

	if err := h.userRepo.Delete(c.Context(), id); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// BanUser godoc
// @Summary      Ban/unban user
// @Description  Toggle user ban status (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /admin/users/{id}/ban [post]
func (h *AdminHandler) BanUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid UUID")
	}

	user, err := h.userRepo.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	user.Status = entity.StatusBanned
	if err := h.userRepo.Update(c.Context(), user); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "user banned", nil)
}

// GetStats godoc
// @Summary      Get system statistics
// @Description  Get system-wide statistics (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /admin/stats [get]
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	ctx := c.Context()

	// Total users by role.
	activeStatus := entity.StatusActive
	users, totalUsers, err := h.userRepo.List(ctx, domrepo.UserFilter{
		Status: &activeStatus, Page: 1, PageSize: 1,
	})
	_ = users
	if err != nil {
		return handleError(c, err)
	}

	// Total files.
	files, totalFiles, err := h.fileRepo.List(ctx, domrepo.FileFilter{Page: 1, PageSize: 1})
	_ = files
	if err != nil {
		return handleError(c, err)
	}

	stats := fiber.Map{
		"total_users": totalUsers,
		"total_files": totalFiles,
		"generated_at": fiber.Map{},
	}
	return response.OK(c, "system statistics", stats)
}
