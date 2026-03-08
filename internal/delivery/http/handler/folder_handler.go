package handler

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/delivery/http/middleware"
	"file-management-service/internal/usecase/folder"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

// FolderHandler handles folder-related HTTP requests.
type FolderHandler struct {
	folderUC  folder.UseCase
	validator *validator.Validator
}

// NewFolderHandler creates a new FolderHandler.
func NewFolderHandler(folderUC folder.UseCase, v *validator.Validator) *FolderHandler {
	return &FolderHandler{folderUC: folderUC, validator: v}
}

// Create godoc
// @Summary      Create folder
// @Description  Create a new folder
// @Tags         folders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      folder.CreateFolderRequest  true  "Folder payload"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /folders [post]
func (h *FolderHandler) Create(c *fiber.Ctx) error {
	var req folder.CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	ownerID := middleware.GetUserID(c)
	result, err := h.folderUC.Create(c.Context(), ownerID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "folder created", result)
}

// List godoc
// @Summary      List folders
// @Description  List folders with optional parent filter
// @Tags         folders
// @Produce      json
// @Security     BearerAuth
// @Param        parent_id  query  string  false  "Parent folder ID"
// @Param        page       query  int     false  "Page" default(1)
// @Param        page_size  query  int     false  "Page size" default(20)
// @Success      200  {object}  response.Response
// @Router       /folders [get]
func (h *FolderHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	parentID := c.Query("parent_id")
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	results, err := h.folderUC.List(c.Context(), userID, parentIDPtr)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "folders retrieved", results)
}

// GetByID godoc
// @Summary      Get folder
// @Description  Get folder metadata by ID
// @Tags         folders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Folder ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /folders/{id} [get]
func (h *FolderHandler) GetByID(c *fiber.Ctx) error {
	folderID := c.Params("id")
	userID := middleware.GetUserID(c)

	result, err := h.folderUC.GetByID(c.Context(), userID, folderID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "folder retrieved", result)
}

// Update godoc
// @Summary      Update folder
// @Description  Update folder metadata
// @Tags         folders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                     true  "Folder ID (UUID)"
// @Param        request  body  folder.UpdateFolderRequest  true  "Update payload"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /folders/{id} [put]
func (h *FolderHandler) Update(c *fiber.Ctx) error {
	folderID := c.Params("id")
	var req folder.UpdateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	result, err := h.folderUC.Update(c.Context(), userID, folderID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "folder updated", result)
}

// Delete godoc
// @Summary      Delete folder
// @Description  Delete a folder and its contents recursively
// @Tags         folders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Folder ID (UUID)"
// @Success      204
// @Failure      403  {object}  response.Response
// @Router       /folders/{id} [delete]
func (h *FolderHandler) Delete(c *fiber.Ctx) error {
	folderID := c.Params("id")
	userID := middleware.GetUserID(c)

	if err := h.folderUC.Delete(c.Context(), userID, folderID); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// Move godoc
// @Summary      Move folder
// @Description  Move a folder to a different parent
// @Tags         folders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                   true  "Folder ID (UUID)"
// @Param        request  body  folder.MoveFolderRequest  true  "Destination parent"
// @Success      200  {object}  response.Response
// @Router       /folders/{id}/move [patch]
func (h *FolderHandler) Move(c *fiber.Ctx) error {
	folderID := c.Params("id")
	var req folder.MoveFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	userID := middleware.GetUserID(c)
	result, err := h.folderUC.Move(c.Context(), userID, folderID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "folder moved", result)
}

// GetTree godoc
// @Summary      Get folder tree
// @Description  Get recursive tree of a folder and its descendants
// @Tags         folders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Folder ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /folders/{id}/tree [get]
func (h *FolderHandler) GetTree(c *fiber.Ctx) error {
	folderID := c.Params("id")
	userID := middleware.GetUserID(c)
	var folderIDPtr *string
	if folderID != "" && folderID != "root" {
		folderIDPtr = &folderID
	}

	result, err := h.folderUC.GetTree(c.Context(), userID, folderIDPtr)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "folder tree retrieved", result)
}

// GetBreadcrumb godoc
// @Summary      Get folder breadcrumb
// @Description  Get breadcrumb path from root to the folder
// @Tags         folders
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Folder ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /folders/{id}/breadcrumb [get]
func (h *FolderHandler) GetBreadcrumb(c *fiber.Ctx) error {
	folderID := c.Params("id")
	userID := middleware.GetUserID(c)

	result, err := h.folderUC.GetBreadcrumb(c.Context(), userID, folderID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "breadcrumb retrieved", result)
}

// Share godoc
// @Summary      Share folder
// @Description  Create a share link for a folder
// @Tags         folders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Folder ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /folders/{id}/share [post]
func (h *FolderHandler) Share(c *fiber.Ctx) error {
	folderID := c.Params("id")
	var req folder.ShareFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	result, err := h.folderUC.Share(c.Context(), userID, folderID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "share link created", result)
}
