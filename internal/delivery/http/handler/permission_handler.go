package handler

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/delivery/http/middleware"
	"file-management-service/internal/usecase/permission"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

type PermissionHandler struct {
	permUC    permission.UseCase
	validator *validator.Validator
}

func NewPermissionHandler(permUC permission.UseCase, v *validator.Validator) *PermissionHandler {
	return &PermissionHandler{permUC: permUC, validator: v}
}

// Grant godoc
// @Summary      Grant permission
// @Description  Grant a user permission on a file or folder
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      permission.GrantPermissionRequest  true  "Grant payload"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /permissions [post]
func (h *PermissionHandler) Grant(c *fiber.Ctx) error {
	var req permission.GrantPermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	grantedByID := middleware.GetUserID(c)
	result, err := h.permUC.Grant(c.Context(), grantedByID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "permission granted", result)
}

// Revoke godoc
// @Summary      Revoke permission
// @Description  Revoke a user's permission on a resource
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Permission ID (UUID)"
// @Success      204
// @Failure      403  {object}  response.Response
// @Router       /permissions/{id} [delete]
func (h *PermissionHandler) Revoke(c *fiber.Ctx) error {
	permID := c.Params("id")
	revokerID := middleware.GetUserID(c)

	if err := h.permUC.Revoke(c.Context(), revokerID, permID); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// List godoc
// @Summary      List permissions for a resource
// @Description  Get all permissions for a specific file or folder
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        resource_type  query  string  true  "Resource type (file|folder)"
// @Param        resource_id    query  string  true  "Resource ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /permissions/resource [get]
func (h *PermissionHandler) List(c *fiber.Ctx) error {
	resourceID := c.Query("resource_id")
	resourceType := c.Query("resource_type")
	if resourceID == "" || resourceType == "" {
		return response.BadRequest(c, "resource_id and resource_type query params are required")
	}

	callerID := middleware.GetUserID(c)
	result, err := h.permUC.List(c.Context(), callerID, resourceID, resourceType)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "permissions retrieved", result)
}

// Check godoc
// @Summary      Check permission
// @Description  Check if the current user has a specific permission on a resource
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  permission.CheckPermissionRequest  true  "Check payload"
// @Success      200  {object}  response.Response
// @Router       /permissions/check [post]
func (h *PermissionHandler) Check(c *fiber.Ctx) error {
	var req permission.CheckPermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	result, err := h.permUC.Check(c.Context(), &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "permission check result", result)
}

// GrantBulk godoc
// @Summary      Bulk grant permissions
// @Description  Grant multiple permissions in a single request
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  permission.GrantBulkRequest  true  "Bulk grant payload"
// @Success      201  {object}  response.Response
// @Router       /permissions/bulk [post]
func (h *PermissionHandler) GrantBulk(c *fiber.Ctx) error {
	var req permission.GrantBulkRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	grantedByID := middleware.GetUserID(c)
	result, err := h.permUC.GrantBulk(c.Context(), grantedByID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "permissions granted in bulk", result)
}
