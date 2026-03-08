package file

import (
	"context"
	"fmt"
	"io"
	"time"

	"file-management-service/config"
	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/crypto"
	"file-management-service/pkg/logger"
	"file-management-service/pkg/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// defaultChunkSize is the recommended size for each upload chunk (5 MB).
const defaultChunkSize int64 = 5 * 1024 * 1024

// chunkUploadTTL is how long an open chunk-upload session lives before expiry.
const chunkUploadTTL = 24 * time.Hour

// StorageService abstracts the object-storage backend.
type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, mimeType string) error
	// Download retrieves a readable stream for a stored object.
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
	// GetPresignedURL returns a time-limited pre-signed URL for direct download.
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	// Copy duplicates an object to a new key within the same bucket.
	Copy(ctx context.Context, srcKey, dstKey string) error
	UploadChunk(ctx context.Context, uploadID, key string, chunkIndex int, reader io.Reader, size int64) error
	// CompleteMultipartUpload assembles parts into the final object.
	CompleteMultipartUpload(ctx context.Context, uploadID, key string, totalChunks int) error
	// AbortMultipartUpload discards all uploaded parts for a session.
	AbortMultipartUpload(ctx context.Context, uploadID, key string) error
}

// WorkerClient enqueues background jobs.
type WorkerClient interface {
	// EnqueueFileProcessing schedules a post-upload processing job.
	EnqueueFileProcessing(ctx context.Context, fileID uuid.UUID, jobType string) error
	// EnqueueNotification schedules an out-of-band notification.
	EnqueueNotification(ctx context.Context, userID uuid.UUID, notifType, message string) error
}

// NotificationService sends real-time notifications to users.
type NotificationService interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title, message string, data map[string]interface{}) error
}

type useCaseImpl struct {
	fileRepo   repository.FileRepository
	folderRepo repository.FolderRepository
	permRepo   repository.PermissionRepository
	userRepo   repository.UserRepository
	auditRepo  repository.AuditRepository
	storage    StorageService
	worker     WorkerClient
	notif      NotificationService
	uploadCfg  *config.UploadConfig
}

func NewUseCase(
	fileRepo repository.FileRepository,
	folderRepo repository.FolderRepository,
	permRepo repository.PermissionRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditRepository,
	storage StorageService,
	worker WorkerClient,
	notif NotificationService,
	uploadCfg *config.UploadConfig,
) UseCase {
	return &useCaseImpl{
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		permRepo:   permRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
		storage:    storage,
		worker:     worker,
		notif:      notif,
		uploadCfg:  uploadCfg,
	}
}

func (uc *useCaseImpl) Upload(
	ctx context.Context,
	ownerID string,
	req *UploadFileRequest,
	filename string,
	fileReader io.Reader,
	fileSize int64,
) (*FileResponse, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, errors.BadRequest("invalid owner ID")
	}

	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}
	if !user.HasStorageSpace(fileSize) {
		return nil, errors.StorageQuotaExceeded()
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return nil, errors.BadRequest("invalid folder ID")
		}
		folderID = &fid
		if _, err := uc.folderRepo.GetByID(ctx, fid); err != nil {
			return nil, errors.NotFound("folder")
		}
	}

	storageKey := utils.GenerateStorageKey(ownerID, filename)
	mimeType := utils.GetMimeType(filename)

	if err := uc.storage.Upload(ctx, storageKey, fileReader, fileSize, mimeType); err != nil {
		logger.Error("storage upload failed", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	fileID := uuid.New()
	now := time.Now()
	f := &entity.File{
		ID:           fileID,
		Name:         utils.SanitizeFilename(filename),
		OriginalName: filename,
		Extension:    utils.GetFileExtension(filename),
		MimeType:     mimeType,
		Size:         fileSize,
		StorageKey:   storageKey,
		StorageBucket: "files",
		FolderID:     folderID,
		OwnerID:      uid,
		Version:      1,
		Status:       entity.FileStatusReady,
		IsPublic:     false,
		Tags:         req.Tags,
		Description:  req.Description,
		Checksum:     "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.fileRepo.Create(ctx, f); err != nil {
		logger.Error("failed to persist file record", zap.Error(err))
		_ = uc.storage.Delete(ctx, storageKey)
		return nil, errors.InternalServer(err)
	}

	if err := uc.userRepo.UpdateStorageUsed(ctx, uid, fileSize); err != nil {
		logger.Warn("failed to update storage used", zap.Error(err))
	}

	go uc.publishFileEvent(context.Background(), uid, fileID, "file_uploaded", "File uploaded", filename)

	return toFileResponse(f), nil
}

func (uc *useCaseImpl) InitChunkUpload(
	ctx context.Context,
	ownerID string,
	req *InitChunkUploadRequest,
) (*ChunkUploadInitResponse, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, errors.BadRequest("invalid owner ID")
	}

	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}
	if !user.HasStorageSpace(req.FileSize) {
		return nil, errors.StorageQuotaExceeded()
	}

	uploadID := uuid.New().String()
	expiresAt := time.Now().Add(chunkUploadTTL)

	return &ChunkUploadInitResponse{
		UploadID:    uploadID,
		ChunkSize:   defaultChunkSize,
		TotalChunks: req.TotalChunks,
		ExpiresAt:   expiresAt,
	}, nil
}

func (uc *useCaseImpl) UploadChunk(
	ctx context.Context,
	ownerID, uploadID string,
	chunkIndex int,
	chunkReader io.Reader,
	chunkSize int64,
	checksum string,
) error {
	if _, err := uuid.Parse(ownerID); err != nil {
		return errors.BadRequest("invalid owner ID")
	}

	chunkKey := fmt.Sprintf("chunks/%s/%d", uploadID, chunkIndex)
	if err := uc.storage.UploadChunk(ctx, uploadID, chunkKey, chunkIndex, chunkReader, chunkSize); err != nil {
		logger.Error("failed to upload chunk", zap.Error(err), zap.String("upload_id", uploadID), zap.Int("chunk", chunkIndex))
		return errors.InternalServer(err)
	}
	return nil
}

func (uc *useCaseImpl) CompleteChunkUpload(
	ctx context.Context,
	ownerID, uploadID string,
) (*FileResponse, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, errors.BadRequest("invalid owner ID")
	}

	storageKey := fmt.Sprintf("assembled/%s/%s", ownerID, uploadID)
	if err := uc.storage.CompleteMultipartUpload(ctx, uploadID, storageKey, 0); err != nil {
		logger.Error("failed to assemble chunks", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	now := time.Now()
	fileID := uuid.New()
	f := &entity.File{
		ID:           fileID,
		Name:         uploadID,
		OriginalName: uploadID,
		StorageKey:   storageKey,
		StorageBucket: "files",
		OwnerID:      uid,
		Version:      1,
		Status:       entity.FileStatusProcessing,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.fileRepo.Create(ctx, f); err != nil {
		logger.Error("failed to create file record after chunk assembly", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	if uc.worker != nil {
		_ = uc.worker.EnqueueFileProcessing(ctx, fileID, "process_chunked_upload")
	}

	return toFileResponse(f), nil
}

func (uc *useCaseImpl) Download(
	ctx context.Context,
	userID, fileID string,
) (io.ReadCloser, *FileResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionDownload)
	if err != nil {
		return nil, nil, err
	}

	stream, _, err := uc.storage.Download(ctx, f.StorageKey)
	if err != nil {
		logger.Error("storage download failed", zap.Error(err))
		return nil, nil, errors.InternalServer(err)
	}

	uid, _ := uuid.Parse(fileID)
	if err := uc.fileRepo.IncrementDownloadCount(ctx, uid); err != nil {
		logger.Warn("failed to increment download count", zap.Error(err))
	}

	return stream, toFileResponse(f), nil
}

func (uc *useCaseImpl) GetPresignedURL(
	ctx context.Context,
	userID, fileID string,
	expiry time.Duration,
) (string, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionDownload)
	if err != nil {
		return "", err
	}

	url, err := uc.storage.GetPresignedURL(ctx, f.StorageKey, expiry)
	if err != nil {
		return "", errors.InternalServer(err)
	}
	return url, nil
}

func (uc *useCaseImpl) Delete(ctx context.Context, userID, fileID string) error {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionDelete)
	if err != nil {
		return err
	}

	if err := uc.storage.Delete(ctx, f.StorageKey); err != nil {
		logger.Error("storage delete failed", zap.Error(err))
		return errors.InternalServer(err)
	}

	fid, _ := uuid.Parse(fileID)
	if err := uc.fileRepo.SoftDelete(ctx, fid); err != nil {
		return errors.InternalServer(err)
	}

	uid, _ := uuid.Parse(userID)
	if err := uc.userRepo.UpdateStorageUsed(ctx, uid, -f.Size); err != nil {
		logger.Warn("failed to reclaim storage quota", zap.Error(err))
	}

	return nil
}

func (uc *useCaseImpl) Move(
	ctx context.Context,
	userID, fileID string,
	req *MoveFileRequest,
) (*FileResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionWrite)
	if err != nil {
		return nil, err
	}

	if req.TargetFolderID != nil {
		targetID, err := uuid.Parse(*req.TargetFolderID)
		if err != nil {
			return nil, errors.BadRequest("invalid target folder ID")
		}
		if _, err := uc.folderRepo.GetByID(ctx, targetID); err != nil {
			return nil, errors.NotFound("target folder")
		}
		f.FolderID = &targetID
	} else {
		f.FolderID = nil
	}

	f.UpdatedAt = time.Now()
	if err := uc.fileRepo.Update(ctx, f); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toFileResponse(f), nil
}

func (uc *useCaseImpl) Rename(
	ctx context.Context,
	userID, fileID string,
	req *RenameFileRequest,
) (*FileResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionWrite)
	if err != nil {
		return nil, err
	}

	f.Name = utils.SanitizeFilename(req.Name)
	f.UpdatedAt = time.Now()
	if err := uc.fileRepo.Update(ctx, f); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toFileResponse(f), nil
}

func (uc *useCaseImpl) Copy(
	ctx context.Context,
	userID, fileID string,
	targetFolderID *string,
) (*FileResponse, error) {
	src, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionRead)
	if err != nil {
		return nil, err
	}

	uid, _ := uuid.Parse(userID)
	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.NotFound("user")
	}
	if !user.HasStorageSpace(src.Size) {
		return nil, errors.StorageQuotaExceeded()
	}

	dstKey := utils.GenerateStorageKey(userID, src.OriginalName)
	if err := uc.storage.Copy(ctx, src.StorageKey, dstKey); err != nil {
		return nil, errors.InternalServer(err)
	}

	var folderID *uuid.UUID
	if targetFolderID != nil {
		fid, err := uuid.Parse(*targetFolderID)
		if err != nil {
			return nil, errors.BadRequest("invalid target folder ID")
		}
		folderID = &fid
	}

	now := time.Now()
	newFile := &entity.File{
		ID:            uuid.New(),
		Name:          src.Name,
		OriginalName:  src.OriginalName,
		Extension:     src.Extension,
		MimeType:      src.MimeType,
		Size:          src.Size,
		StorageKey:    dstKey,
		StorageBucket: src.StorageBucket,
		FolderID:      folderID,
		OwnerID:       uid,
		Version:       1,
		Status:        entity.FileStatusReady,
		Tags:          src.Tags,
		Description:   src.Description,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := uc.fileRepo.Create(ctx, newFile); err != nil {
		_ = uc.storage.Delete(ctx, dstKey)
		return nil, errors.InternalServer(err)
	}

	if err := uc.userRepo.UpdateStorageUsed(ctx, uid, src.Size); err != nil {
		logger.Warn("failed to update storage used after copy", zap.Error(err))
	}

	return toFileResponse(newFile), nil
}

func (uc *useCaseImpl) GetByID(ctx context.Context, userID, fileID string) (*FileResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionRead)
	if err != nil {
		return nil, err
	}
	return toFileResponse(f), nil
}

func (uc *useCaseImpl) List(
	ctx context.Context,
	userID string,
	folderID *string,
	filter *FileListFilter,
) ([]*FileResponse, int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, errors.BadRequest("invalid user ID")
	}

	var fid *uuid.UUID
	if folderID != nil {
		parsed, err := uuid.Parse(*folderID)
		if err != nil {
			return nil, 0, errors.BadRequest("invalid folder ID")
		}
		fid = &parsed
	}

	repoFilter := toRepoFileFilter(uid, fid, filter)
	files, total, err := uc.fileRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, errors.InternalServer(err)
	}

	resp := make([]*FileResponse, len(files))
	for i, f := range files {
		resp[i] = toFileResponse(f)
	}
	return resp, total, nil
}

func (uc *useCaseImpl) Search(ctx context.Context, userID, query string) ([]*FileResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	filter := repository.FileFilter{OwnerID: &uid, Search: query, Page: 1, PageSize: 50}
	files, _, err := uc.fileRepo.Search(ctx, query, filter)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	resp := make([]*FileResponse, len(files))
	for i, f := range files {
		resp[i] = toFileResponse(f)
	}
	return resp, nil
}

func (uc *useCaseImpl) Share(
	ctx context.Context,
	userID, fileID string,
	req *ShareFileRequest,
) (*ShareLinkResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionShare)
	if err != nil {
		return nil, err
	}

	uid, _ := uuid.Parse(userID)
	fid, _ := uuid.Parse(fileID)

	token, err := crypto.GenerateSecureToken(24)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	var hashedPassword *string
	if req.Password != nil {
		h, err := crypto.HashPassword(*req.Password)
		if err != nil {
			return nil, errors.InternalServer(err)
		}
		hashedPassword = &h
	}

	now := time.Now()
	link := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceID:   fid,
		ResourceType: entity.ResourceTypeFile,
		CreatedByID:  uid,
		Action:       entity.PermissionAction(req.Action),
		ExpiresAt:    req.ExpiresAt,
		MaxUses:      req.MaxUses,
		UseCount:     0,
		Password:     hashedPassword,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.permRepo.CreateShareLink(ctx, link); err != nil {
		return nil, errors.InternalServer(err)
	}

	go uc.publishFileEvent(context.Background(), uid, f.ID, "file_shared", "File shared", f.Name)

	return &ShareLinkResponse{
		Token:     token,
		URL:       fmt.Sprintf("/api/v1/share/%s", token),
		ExpiresAt: req.ExpiresAt,
		MaxUses:   req.MaxUses,
		Action:    req.Action,
		CreatedAt: now,
	}, nil
}

func (uc *useCaseImpl) GetVersions(
	ctx context.Context,
	userID, fileID string,
) ([]*FileVersionResponse, error) {
	_, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionRead)
	if err != nil {
		return nil, err
	}

	fid, _ := uuid.Parse(fileID)
	versions, err := uc.fileRepo.GetVersions(ctx, fid)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	resp := make([]*FileVersionResponse, len(versions))
	for i, v := range versions {
		resp[i] = toFileVersionResponse(v)
	}
	return resp, nil
}

func (uc *useCaseImpl) RestoreVersion(
	ctx context.Context,
	userID, fileID string,
	version int,
) (*FileResponse, error) {
	f, err := uc.resolveFileWithAccess(ctx, userID, fileID, entity.ActionWrite)
	if err != nil {
		return nil, err
	}

	fid, _ := uuid.Parse(fileID)
	versions, err := uc.fileRepo.GetVersions(ctx, fid)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	var target *entity.FileVersion
	for _, v := range versions {
		if v.Version == version {
			target = v
			break
		}
	}
	if target == nil {
		return nil, errors.NotFound("file version")
	}

	// Save current state as a new version before restoring.
	uid, _ := uuid.Parse(userID)
	currentVersion := &entity.FileVersion{
		ID:          uuid.New(),
		FileID:      fid,
		Version:     f.Version,
		StorageKey:  f.StorageKey,
		Size:        f.Size,
		Checksum:    f.Checksum,
		ChangedByID: uid,
		CreatedAt:   time.Now(),
	}
	if err := uc.fileRepo.CreateVersion(ctx, currentVersion); err != nil {
		logger.Warn("failed to snapshot version before restore", zap.Error(err))
	}

	f.StorageKey = target.StorageKey
	f.Size = target.Size
	f.Checksum = target.Checksum
	f.Version = f.Version + 1
	f.UpdatedAt = time.Now()

	if err := uc.fileRepo.Update(ctx, f); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toFileResponse(f), nil
}

// resolveFileWithAccess fetches a file and verifies the caller has the required
// permission (owner always has full access).
func (uc *useCaseImpl) resolveFileWithAccess(
	ctx context.Context,
	userID, fileID string,
	action entity.PermissionAction,
) (*entity.File, error) {
	fid, err := uuid.Parse(fileID)
	if err != nil {
		return nil, errors.BadRequest("invalid file ID")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	f, err := uc.fileRepo.GetByID(ctx, fid)
	if err != nil || f == nil {
		return nil, errors.NotFound("file")
	}

	// Owner has full access.
	if f.OwnerID == uid {
		return f, nil
	}

	allowed, err := uc.permRepo.HasPermission(ctx, uid, fid, entity.ResourceTypeFile, action)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	if !allowed {
		return nil, errors.Forbidden(string(action))
	}
	return f, nil
}

// publishFileEvent fires a notification asynchronously; errors are only logged.
func (uc *useCaseImpl) publishFileEvent(
	ctx context.Context,
	userID, fileID uuid.UUID,
	eventType, title, detail string,
) {
	if uc.notif == nil {
		return
	}
	if err := uc.notif.Send(ctx, userID, eventType, title, detail,
		map[string]interface{}{"file_id": fileID.String()}); err != nil {
		logger.Warn("failed to send notification", zap.Error(err), zap.String("type", eventType))
	}
}

func toRepoFileFilter(ownerID uuid.UUID, folderID *uuid.UUID, f *FileListFilter) repository.FileFilter {
	filter := repository.FileFilter{
		OwnerID:  &ownerID,
		FolderID: folderID,
		Page:     1,
		PageSize: 20,
	}
	if f == nil {
		return filter
	}
	if f.MimeType != nil {
		filter.MimeType = *f.MimeType
	}
	if f.Status != nil {
		s := entity.FileStatus(*f.Status)
		filter.Status = &s
	}
	filter.Tags = f.Tags
	filter.Search = f.Search
	filter.OrderBy = f.SortBy
	filter.OrderDir = f.SortOrder
	if f.Page > 0 {
		filter.Page = f.Page
	}
	if f.PageSize > 0 {
		filter.PageSize = f.PageSize
	}
	return filter
}

func toFileResponse(f *entity.File) *FileResponse {
	resp := &FileResponse{
		ID:            f.ID.String(),
		Name:          f.Name,
		OriginalName:  f.OriginalName,
		Extension:     f.Extension,
		MimeType:      f.MimeType,
		Size:          f.Size,
		SizeFormatted: utils.FormatFileSize(f.Size),
		OwnerID:       f.OwnerID.String(),
		Version:       f.Version,
		Status:        string(f.Status),
		IsPublic:      f.IsPublic,
		DownloadCount: f.DownloadCount,
		Tags:          []string(f.Tags),
		Description:   f.Description,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
	}
	if f.FolderID != nil {
		s := f.FolderID.String()
		resp.FolderID = &s
	}
	return resp
}

func toFileVersionResponse(v *entity.FileVersion) *FileVersionResponse {
	return &FileVersionResponse{
		ID:         v.ID.String(),
		FileID:     v.FileID.String(),
		Version:    v.Version,
		Size:       v.Size,
		Checksum:   v.Checksum,
		CreatedBy:  v.ChangedByID.String(),
		ChangeNote: v.ChangeNote,
		CreatedAt:  v.CreatedAt,
	}
}

func (uc *useCaseImpl) DownloadByShareToken(ctx context.Context, token string) (io.ReadCloser, *FileResponse, error) {
	shareLink, err := uc.permRepo.GetShareLinkByToken(ctx, token)
	if err != nil || shareLink == nil {
		return nil, nil, errors.New(410, "share link has expired or reached its usage limit")
	}

	if !shareLink.IsUsable() {
		return nil, nil, errors.New(410, "share link has expired or reached its usage limit")
	}

	if shareLink.ResourceType != entity.ResourceTypeFile {
		return nil, nil, errors.BadRequest("share link does not point to a file")
	}

	file, err := uc.fileRepo.GetByID(ctx, shareLink.ResourceID)
	if err != nil || file == nil {
		return nil, nil, errors.NotFound("file")
	}

	if !file.IsReady() {
		return nil, nil, errors.New(409, "file is not ready for download")
	}

	reader, _, err := uc.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, errors.InternalServer(err)
	}

	shareLink.UseCount++
	if err := uc.permRepo.UpdateShareLink(ctx, shareLink); err != nil {
		logger.Warn("failed to update share link use count", zap.Error(err))
	}

	if err := uc.fileRepo.IncrementDownloadCount(ctx, file.ID); err != nil {
		logger.Warn("failed to increment download count", zap.Error(err))
	}

	return reader, toFileResponse(file), nil
}
