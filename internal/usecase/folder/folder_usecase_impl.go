package folder

import (
	"context"
	"fmt"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/crypto"
	"file-management-service/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationSender is a minimal interface for sending notifications from the folder use case.
type NotificationSender interface {
	Publish(ctx context.Context, event *entity.NotificationEvent) error
}

type useCaseImpl struct {
	folderRepo repository.FolderRepository
	permRepo   repository.PermissionRepository
	auditRepo  repository.AuditRepository
	notif      NotificationSender
}

// NewUseCase constructs the folder UseCase implementation.
func NewUseCase(
	folderRepo repository.FolderRepository,
	permRepo repository.PermissionRepository,
	auditRepo repository.AuditRepository,
	notif NotificationSender,
) UseCase {
	return &useCaseImpl{
		folderRepo: folderRepo,
		permRepo:   permRepo,
		auditRepo:  auditRepo,
		notif:      notif,
	}
}

// Create makes a new folder under an optional parent.
func (uc *useCaseImpl) Create(
	ctx context.Context,
	ownerID string,
	req *CreateFolderRequest,
) (*FolderResponse, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, errors.BadRequest("invalid owner ID")
	}

	var parentID *uuid.UUID
	parentPath := ""
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, errors.BadRequest("invalid parent folder ID")
		}
		parent, err := uc.folderRepo.GetByID(ctx, pid)
		if err != nil || parent == nil {
			return nil, errors.NotFound("parent folder")
		}
		if parent.OwnerID != uid {
			allowed, _ := uc.permRepo.HasPermission(ctx, uid, pid, entity.ResourceTypeFolder, entity.ActionWrite)
			if !allowed {
				return nil, errors.Forbidden("create folder here")
			}
		}
		parentID = &pid
		parentPath = parent.Path
	}

	now := time.Now()
	folderID := uuid.New()
	path := fmt.Sprintf("%s/%s", parentPath, folderID.String())

	folder := &entity.Folder{
		ID:          folderID,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    parentID,
		OwnerID:     uid,
		Path:        path,
		Color:       req.Color,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.folderRepo.Create(ctx, folder); err != nil {
		logger.Error("failed to create folder", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	if parentID != nil {
		_ = uc.folderRepo.UpdateCounts(ctx, *parentID, 0, 1, 0)
	}

	return toFolderResponse(folder, 0, 0), nil
}

// GetByID returns a folder the caller has at least read access to.
func (uc *useCaseImpl) GetByID(
	ctx context.Context,
	userID, folderID string,
) (*FolderResponse, error) {
	folder, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionRead)
	if err != nil {
		return nil, err
	}
	return toFolderResponse(folder, folder.FolderCount, folder.FileCount), nil
}

// Update applies partial changes to a folder.
func (uc *useCaseImpl) Update(
	ctx context.Context,
	userID, folderID string,
	req *UpdateFolderRequest,
) (*FolderResponse, error) {
	folder, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionWrite)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		folder.Name = *req.Name
	}
	if req.Description != nil {
		folder.Description = req.Description
	}
	if req.Color != nil {
		folder.Color = req.Color
	}
	folder.UpdatedAt = time.Now()

	if err := uc.folderRepo.Update(ctx, folder); err != nil {
		return nil, errors.InternalServer(err)
	}
	return toFolderResponse(folder, folder.FolderCount, folder.FileCount), nil
}

// Delete recursively removes a folder and all its contents.
func (uc *useCaseImpl) Delete(ctx context.Context, userID, folderID string) error {
	folder, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionDelete)
	if err != nil {
		return err
	}

	fid, _ := uuid.Parse(folderID)
	if err := uc.folderRepo.SoftDelete(ctx, fid); err != nil {
		return errors.InternalServer(err)
	}

	// Decrement parent's counts.
	if folder.ParentID != nil {
		_ = uc.folderRepo.UpdateCounts(ctx, *folder.ParentID, 0, -1, -folder.Size)
	}
	return nil
}

// Move relocates a folder under a new parent.
func (uc *useCaseImpl) Move(
	ctx context.Context,
	userID, folderID string,
	req *MoveFolderRequest,
) (*FolderResponse, error) {
	folder, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionWrite)
	if err != nil {
		return nil, err
	}

	var newParentID *uuid.UUID
	if req.TargetParentID != nil {
		pid, err := uuid.Parse(*req.TargetParentID)
		if err != nil {
			return nil, errors.BadRequest("invalid target parent ID")
		}
		if _, err := uc.folderRepo.GetByID(ctx, pid); err != nil {
			return nil, errors.NotFound("target folder")
		}
		newParentID = &pid
	}

	fid, _ := uuid.Parse(folderID)
	if err := uc.folderRepo.Move(ctx, fid, newParentID); err != nil {
		return nil, errors.InternalServer(err)
	}

	// Update counters on old and new parent.
	if folder.ParentID != nil {
		_ = uc.folderRepo.UpdateCounts(ctx, *folder.ParentID, 0, -1, -folder.Size)
	}
	if newParentID != nil {
		_ = uc.folderRepo.UpdateCounts(ctx, *newParentID, 0, 1, folder.Size)
	}

	folder.ParentID = newParentID
	folder.UpdatedAt = time.Now()
	return toFolderResponse(folder, folder.FolderCount, folder.FileCount), nil
}

// List returns the immediate children of a folder.
func (uc *useCaseImpl) List(
	ctx context.Context,
	userID string,
	parentID *string,
) ([]*FolderResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	filter := repository.FolderFilter{OwnerID: &uid, Page: 1, PageSize: 100}
	if parentID != nil {
		pid, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, errors.BadRequest("invalid parent folder ID")
		}
		filter.ParentID = &pid
	} else {
		isRoot := true
		filter.IsRoot = &isRoot
	}

	folders, _, err := uc.folderRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	resp := make([]*FolderResponse, len(folders))
	for i, f := range folders {
		resp[i] = toFolderResponse(f, f.FolderCount, f.FileCount)
	}
	return resp, nil
}

// GetTree returns the full recursive subtree.
func (uc *useCaseImpl) GetTree(
	ctx context.Context,
	userID string,
	folderID *string,
) (*FolderTreeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	allFolders, err := uc.folderRepo.GetByOwner(ctx, uid)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	var rootID *uuid.UUID
	if folderID != nil {
		fid, err := uuid.Parse(*folderID)
		if err != nil {
			return nil, errors.BadRequest("invalid folder ID")
		}
		rootID = &fid
	}

	return buildTree(allFolders, rootID), nil
}

// GetBreadcrumb returns the ancestor chain from root to folderID.
func (uc *useCaseImpl) GetBreadcrumb(
	ctx context.Context,
	userID, folderID string,
) ([]*BreadcrumbItem, error) {
	_, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionRead)
	if err != nil {
		return nil, err
	}

	fid, _ := uuid.Parse(folderID)
	chain := []*BreadcrumbItem{}

	current := &fid
	for current != nil {
		f, err := uc.folderRepo.GetByID(ctx, *current)
		if err != nil || f == nil {
			break
		}
		chain = append([]*BreadcrumbItem{{
			ID:   f.ID.String(),
			Name: f.Name,
			Path: f.Path,
		}}, chain...)
		current = f.ParentID
	}
	return chain, nil
}

// Share creates a share link for a folder.
func (uc *useCaseImpl) Share(
	ctx context.Context,
	userID, folderID string,
	req *ShareFolderRequest,
) (*ShareLinkResponse, error) {
	folder, err := uc.resolveFolderWithAccess(ctx, userID, folderID, entity.ActionShare)
	if err != nil {
		return nil, err
	}

	uid, _ := uuid.Parse(userID)
	fid, _ := uuid.Parse(folderID)

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
		ResourceType: entity.ResourceTypeFolder,
		CreatedByID:  uid,
		Action:       entity.PermissionAction(req.Action),
		ExpiresAt:    req.ExpiresAt,
		MaxUses:      req.MaxUses,
		Password:     hashedPassword,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.permRepo.CreateShareLink(ctx, link); err != nil {
		return nil, errors.InternalServer(err)
	}

	_ = folder // used for access check above
	return &ShareLinkResponse{
		Token:     token,
		URL:       fmt.Sprintf("/api/v1/share/%s", token),
		ExpiresAt: req.ExpiresAt,
		MaxUses:   req.MaxUses,
		Action:    req.Action,
		CreatedAt: now,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (uc *useCaseImpl) resolveFolderWithAccess(
	ctx context.Context,
	userID, folderID string,
	action entity.PermissionAction,
) (*entity.Folder, error) {
	fid, err := uuid.Parse(folderID)
	if err != nil {
		return nil, errors.BadRequest("invalid folder ID")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}

	folder, err := uc.folderRepo.GetByID(ctx, fid)
	if err != nil || folder == nil {
		return nil, errors.NotFound("folder")
	}

	if folder.OwnerID == uid {
		return folder, nil
	}

	allowed, err := uc.permRepo.HasPermission(ctx, uid, fid, entity.ResourceTypeFolder, action)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	if !allowed {
		return nil, errors.Forbidden(string(action))
	}
	return folder, nil
}

// buildTree constructs a recursive FolderTreeResponse from a flat list.
func buildTree(all []*entity.Folder, rootID *uuid.UUID) *FolderTreeResponse {
	index := make(map[uuid.UUID]*FolderTreeResponse, len(all))
	for _, f := range all {
		index[f.ID] = &FolderTreeResponse{
			FolderResponse: *toFolderResponse(f, f.FolderCount, f.FileCount),
		}
	}

	var root *FolderTreeResponse
	for _, f := range all {
		node := index[f.ID]
		if f.ParentID != nil {
			if parent, ok := index[*f.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		} else if rootID == nil {
			root = node
		}
		if rootID != nil && f.ID == *rootID {
			root = node
		}
	}

	if root == nil {
		root = &FolderTreeResponse{}
	}
	return root
}

// toFolderResponse converts a domain Folder entity to the public FolderResponse DTO.
func toFolderResponse(f *entity.Folder, childrenCount, fileCount int64) *FolderResponse {
	resp := &FolderResponse{
		ID:            f.ID.String(),
		Name:          f.Name,
		Description:   f.Description,
		OwnerID:       f.OwnerID.String(),
		Path:          f.Path,
		IsRoot:        f.IsRoot,
		IsShared:      f.IsShared,
		Color:         f.Color,
		ChildrenCount: childrenCount,
		FileCount:     fileCount,
		Size:          f.Size,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
	}
	if f.ParentID != nil {
		s := f.ParentID.String()
		resp.ParentID = &s
	}
	return resp
}
