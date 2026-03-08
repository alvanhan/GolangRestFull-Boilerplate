package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"file-management-service/config"
	"file-management-service/internal/delivery/http/middleware"
	"file-management-service/internal/usecase/file"
	"file-management-service/pkg/pagination"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

// FileHandler handles file-related HTTP requests.
type FileHandler struct {
	fileUC    file.UseCase
	validator *validator.Validator
	uploadCfg *config.UploadConfig
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(fileUC file.UseCase, v *validator.Validator, uploadCfg *config.UploadConfig) *FileHandler {
	return &FileHandler{fileUC: fileUC, validator: v, uploadCfg: uploadCfg}
}

// Upload godoc
// @Summary      Upload a file
// @Description  Upload a file (multipart/form-data). Max size controlled by config.
// @Tags         files
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        file       formData  file    true  "File to upload"
// @Param        folder_id  formData  string  false "Target folder ID (UUID)"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      413  {object}  response.Response
// @Router       /files/upload [post]
func (h *FileHandler) Upload(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file field is required in multipart form")
	}

	if fh.Size > h.uploadCfg.MaxSize {
		return response.Error(c, fiber.StatusRequestEntityTooLarge,
			fmt.Sprintf("file too large, maximum allowed size is %d bytes", h.uploadCfg.MaxSize), nil)
	}

	f, err := fh.Open()
	if err != nil {
		return response.InternalError(c)
	}
	defer f.Close()

	var req file.UploadFileRequest
	_ = c.BodyParser(&req)

	ownerID := middleware.GetUserID(c)
	result, err := h.fileUC.Upload(c.Context(), ownerID, &req, fh.Filename, f, fh.Size)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "file uploaded successfully", result)
}

// InitChunkUpload godoc
// @Summary      Initialize chunked upload
// @Description  Start a multipart/chunked upload session for large files
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      file.InitChunkUploadRequest  true  "Init chunk upload"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /files/upload/init [post]
func (h *FileHandler) InitChunkUpload(c *fiber.Ctx) error {
	var req file.InitChunkUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	ownerID := middleware.GetUserID(c)
	result, err := h.fileUC.InitChunkUpload(c.Context(), ownerID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "chunk upload session created", result)
}

// UploadChunk godoc
// @Summary      Upload a file chunk
// @Description  Upload a single chunk of a previously initiated multipart upload
// @Tags         files
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        upload_id    formData  string  true  "Upload session ID"
// @Param        chunk_index  formData  int     true  "Chunk index (0-based)"
// @Param        chunk        formData  file    true  "Chunk data"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /files/upload/chunk [post]
func (h *FileHandler) UploadChunk(c *fiber.Ctx) error {
	fh, err := c.FormFile("chunk")
	if err != nil {
		return response.BadRequest(c, "chunk field is required in multipart form")
	}

	uploadID := c.FormValue("upload_id")
	if uploadID == "" {
		return response.BadRequest(c, "upload_id is required")
	}
	chunkIndexStr := c.FormValue("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		return response.BadRequest(c, "chunk_index must be an integer")
	}
	checksum := c.FormValue("checksum")
	if checksum == "" {
		return response.BadRequest(c, "checksum is required")
	}

	f, err := fh.Open()
	if err != nil {
		return response.InternalError(c)
	}
	defer f.Close()

	ownerID := middleware.GetUserID(c)
	if err := h.fileUC.UploadChunk(c.Context(), ownerID, uploadID, chunkIndex, f, fh.Size, checksum); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "chunk uploaded", nil)
}

// CompleteChunkUpload godoc
// @Summary      Complete chunked upload
// @Description  Finalize a multipart upload by assembling all chunks
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      object  true  "Complete chunk upload (upload_id required)"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /files/upload/complete [post]
func (h *FileHandler) CompleteChunkUpload(c *fiber.Ctx) error {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		var body struct {
			UploadID string `json:"upload_id"`
		}
		_ = c.BodyParser(&body)
		uploadID = body.UploadID
	}
	if uploadID == "" {
		return response.BadRequest(c, "upload_id is required")
	}

	ownerID := middleware.GetUserID(c)
	result, err := h.fileUC.CompleteChunkUpload(c.Context(), ownerID, uploadID)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "file assembled successfully", result)
}

// List godoc
// @Summary      List files
// @Description  List files in a folder with pagination and filtering
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        folder_id   query  string  false  "Folder ID filter"
// @Param        page        query  int     false  "Page number"  default(1)
// @Param        page_size   query  int     false  "Items per page" default(20)
// @Param        sort_by     query  string  false  "Sort field (name|size|created_at|updated_at)"
// @Param        sort_order  query  string  false  "Sort order (asc|desc)"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /files [get]
func (h *FileHandler) List(c *fiber.Ctx) error {
	pag, err := pagination.ParseFromQuery(c)
	if err != nil {
		return response.BadRequest(c, "invalid pagination parameters")
	}

	folderID := c.Query("folder_id")
	var folderIDPtr *string
	if folderID != "" {
		folderIDPtr = &folderID
	}

	filter := &file.FileListFilter{
		Page:      pag.Page,
		PageSize:  pag.PageSize,
		SortBy:    c.Query("order_by", "created_at"),
		SortOrder: c.Query("order_dir", "desc"),
	}
	if mt := c.Query("mime_type"); mt != "" {
		filter.MimeType = &mt
	}
	if st := c.Query("status"); st != "" {
		filter.Status = &st
	}

	userID := middleware.GetUserID(c)
	results, total, err := h.fileUC.List(c.Context(), userID, folderIDPtr, filter)
	if err != nil {
		return handleError(c, err)
	}

	meta := pagination.NewMeta(pag.Page, pag.PageSize, total)
	return response.SuccessWithMeta(c, fiber.StatusOK, "files retrieved", results, meta)
}

// GetByID godoc
// @Summary      Get file by ID
// @Description  Get file metadata by UUID
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "File ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /files/{id} [get]
func (h *FileHandler) GetByID(c *fiber.Ctx) error {
	fileID := c.Params("id")
	userID := middleware.GetUserID(c)

	result, err := h.fileUC.GetByID(c.Context(), userID, fileID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "file retrieved", result)
}

// Download godoc
// @Summary      Download a file
// @Description  Download file content by ID
// @Tags         files
// @Produce      octet-stream
// @Security     BearerAuth
// @Param        id  path  string  true  "File ID (UUID)"
// @Success      200
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /files/{id}/download [get]
func (h *FileHandler) Download(c *fiber.Ctx) error {
	fileID := c.Params("id")
	userID := middleware.GetUserID(c)

	reader, meta, err := h.fileUC.Download(c.Context(), userID, fileID)
	if err != nil {
		return handleError(c, err)
	}
	defer reader.Close()

	c.Set(fiber.HeaderContentType, meta.MimeType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, meta.OriginalName))
	c.Set(fiber.HeaderContentLength, strconv.FormatInt(meta.Size, 10))

	return c.SendStream(reader, int(meta.Size))
}

// GetPresignedURL godoc
// @Summary      Get presigned download URL
// @Description  Get a temporary presigned URL to download the file directly from storage
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "File ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/presigned [get]
func (h *FileHandler) GetPresignedURL(c *fiber.Ctx) error {
	fileID := c.Params("id")
	userID := middleware.GetUserID(c)

	expiryMinutes, _ := strconv.ParseInt(c.Query("expiry_minutes", "60"), 10, 64)
	if expiryMinutes <= 0 || expiryMinutes > 1440 {
		expiryMinutes = 60
	}
	expiry := time.Duration(expiryMinutes) * time.Minute

	url, err := h.fileUC.GetPresignedURL(c.Context(), userID, fileID, expiry)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "presigned URL generated", fiber.Map{"url": url})
}

// Move godoc
// @Summary      Move a file
// @Description  Move a file to a different folder
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string          true  "File ID (UUID)"
// @Param        request  body  file.MoveFileRequest  true  "Destination folder"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/move [patch]
func (h *FileHandler) Move(c *fiber.Ctx) error {
	fileID := c.Params("id")
	var req file.MoveFileRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	userID := middleware.GetUserID(c)
	result, err := h.fileUC.Move(c.Context(), userID, fileID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "file moved", result)
}

// Rename godoc
// @Summary      Rename a file
// @Description  Rename a file
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string            true  "File ID (UUID)"
// @Param        request  body  file.RenameFileRequest  true  "New file name"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/rename [patch]
func (h *FileHandler) Rename(c *fiber.Ctx) error {
	fileID := c.Params("id")
	var req file.RenameFileRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	result, err := h.fileUC.Rename(c.Context(), userID, fileID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "file renamed", result)
}

// Copy godoc
// @Summary      Copy a file
// @Description  Copy a file to a different folder
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string          true  "File ID (UUID)"
// @Param        request  body  object  true  "Destination folder (target_folder_id)"
// @Success      201  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/copy [post]
func (h *FileHandler) Copy(c *fiber.Ctx) error {
	fileID := c.Params("id")
	var body struct {
		TargetFolderID *string `json:"target_folder_id"`
	}
	_ = c.BodyParser(&body)

	userID := middleware.GetUserID(c)
	result, err := h.fileUC.Copy(c.Context(), userID, fileID, body.TargetFolderID)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "file copied", result)
}

// Delete godoc
// @Summary      Delete a file
// @Description  Soft-delete a file
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "File ID (UUID)"
// @Success      204
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /files/{id} [delete]
func (h *FileHandler) Delete(c *fiber.Ctx) error {
	fileID := c.Params("id")
	userID := middleware.GetUserID(c)

	if err := h.fileUC.Delete(c.Context(), userID, fileID); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// Share godoc
// @Summary      Create share link
// @Description  Generate a public share link for a file
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string            true  "File ID (UUID)"
// @Param        request  body  file.ShareFileRequest   true  "Share options"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/share [post]
func (h *FileHandler) Share(c *fiber.Ctx) error {
	fileID := c.Params("id")
	var req file.ShareFileRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if errs := h.validator.ValidateStruct(req); len(errs) > 0 {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation failed", errs)
	}

	userID := middleware.GetUserID(c)
	result, err := h.fileUC.Share(c.Context(), userID, fileID, &req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Created(c, "share link created", result)
}

// GetVersions godoc
// @Summary      Get file versions
// @Description  List all versions of a file
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "File ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/versions [get]
func (h *FileHandler) GetVersions(c *fiber.Ctx) error {
	fileID := c.Params("id")
	userID := middleware.GetUserID(c)

	result, err := h.fileUC.GetVersions(c.Context(), userID, fileID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "versions retrieved", result)
}

// RestoreVersion godoc
// @Summary      Restore file version
// @Description  Restore a file to a previous version
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "File ID (UUID)"
// @Param        ver  path  int     true  "Version number"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/{id}/versions/{ver}/restore [post]
func (h *FileHandler) RestoreVersion(c *fiber.Ctx) error {
	fileID := c.Params("id")
	version, err := strconv.Atoi(c.Params("ver"))
	if err != nil {
		return response.BadRequest(c, "invalid version number")
	}

	userID := middleware.GetUserID(c)
	result, err := h.fileUC.RestoreVersion(c.Context(), userID, fileID, version)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "version restored", result)
}

// Search godoc
// @Summary      Search files
// @Description  Full-text search across file names and metadata
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        q          query  string  true   "Search query"
// @Param        page       query  int     false  "Page number" default(1)
// @Param        page_size  query  int     false  "Items per page" default(20)
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /files/search [get]
func (h *FileHandler) Search(c *fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return response.BadRequest(c, "query parameter 'q' is required")
	}

	userID := middleware.GetUserID(c)
	results, err := h.fileUC.Search(c.Context(), userID, q)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "search results", results)
}

// DownloadByShareToken godoc
// @Summary      Download via share token
// @Description  Download a file using a public share token (no auth required)
// @Tags         files
// @Produce      octet-stream
// @Param        token  path  string  true  "Share token"
// @Success      200
// @Failure      404  {object}  response.Response
// @Router       /share/{token} [get]
func (h *FileHandler) DownloadByShareToken(c *fiber.Ctx) error {
	token := c.Params("token")

	reader, meta, err := h.fileUC.DownloadByShareToken(c.Context(), token)
	if err != nil {
		return handleError(c, err)
	}
	defer reader.Close()

	c.Set(fiber.HeaderContentType, meta.MimeType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, meta.OriginalName))
	c.Set(fiber.HeaderContentLength, strconv.FormatInt(meta.Size, 10))

	return c.SendStream(reader, int(meta.Size))
}
