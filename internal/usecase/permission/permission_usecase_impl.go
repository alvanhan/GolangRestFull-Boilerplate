package permission

import (
	"context"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"
	"file-management-service/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationSender is a minimal interface for sending notifications from the permission use case.
type NotificationSender interface {
	Publish(ctx context.Context, event *entity.NotificationEvent) error
}

type useCaseImpl struct {
	permRepo   repository.PermissionRepository
	fileRepo   repository.FileRepository
	folderRepo repository.FolderRepository
	auditRepo  repository.AuditRepository
	notif      NotificationSender
}

func NewUseCase(
	permRepo repository.PermissionRepository,
	fileRepo repository.FileRepository,
	folderRepo repository.FolderRepository,
	auditRepo repository.AuditRepository,
	notif NotificationSender,
) UseCase {
	return &useCaseImpl{
		permRepo:   permRepo,
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		auditRepo:  auditRepo,
		notif:      notif,
	}
}

func (uc *useCaseImpl) Grant(
	ctx context.Context,
	grantedByID string,
	req *GrantPermissionRequest,
) (*PermissionResponse, error) {
	granterUID, err := uuid.Parse(grantedByID)
	if err != nil {
		return nil, errors.BadRequest("invalid granter ID")
	}
	targetUID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	resourceUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return nil, errors.BadRequest("invalid resource ID")
	}

	// Verify granter has manage_permissions on this resource.
	resType := entity.ResourceType(req.ResourceType)
	allowed, err := uc.permRepo.HasPermission(ctx, granterUID, resourceUID, resType, entity.ActionManagePermissions)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	if !allowed {
		return nil, errors.Forbidden("manage permissions")
	}

	perm := &entity.Permission{
		ID:           uuid.New(),
		ResourceID:   resourceUID,
		ResourceType: resType,
		UserID:       targetUID,
		Action:       entity.PermissionAction(req.Action),
		GrantedByID:  granterUID,
		ExpiresAt:    req.ExpiresAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.permRepo.Create(ctx, perm); err != nil {
		logger.Error("failed to grant permission", zap.Error(err))
		return nil, errors.InternalServer(err)
	}

	go uc.logAudit(context.Background(), granterUID, resourceUID, req.ResourceType,
		string(entity.AuditPermissionGrant), "success", nil)

	return toPermissionResponse(perm), nil
}

func (uc *useCaseImpl) Revoke(
	ctx context.Context,
	revokerID, permissionID string,
) error {
	revokerUID, err := uuid.Parse(revokerID)
	if err != nil {
		return errors.BadRequest("invalid revoker ID")
	}
	permUID, err := uuid.Parse(permissionID)
	if err != nil {
		return errors.BadRequest("invalid permission ID")
	}

	perm, err := uc.permRepo.GetByID(ctx, permUID)
	if err != nil || perm == nil {
		return errors.NotFound("permission")
	}

	// Only the resource owner or a manage_permissions holder may revoke.
	allowed, _ := uc.permRepo.HasPermission(ctx, revokerUID, perm.ResourceID, perm.ResourceType, entity.ActionManagePermissions)
	if !allowed && perm.GrantedByID != revokerUID {
		return errors.Forbidden("revoke permission")
	}

	if err := uc.permRepo.Delete(ctx, permUID); err != nil {
		return errors.InternalServer(err)
	}

	go uc.logAudit(context.Background(), revokerUID, perm.ResourceID, string(perm.ResourceType),
		string(entity.AuditPermissionRevoke), "success", nil)

	return nil
}

func (uc *useCaseImpl) List(
	ctx context.Context,
	callerID, resourceID, resourceType string,
) (*ListPermissionsResponse, error) {
	callerUID, err := uuid.Parse(callerID)
	if err != nil {
		return nil, errors.BadRequest("invalid caller ID")
	}
	resourceUID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, errors.BadRequest("invalid resource ID")
	}

	resType := entity.ResourceType(resourceType)
	allowed, _ := uc.permRepo.HasPermission(ctx, callerUID, resourceUID, resType, entity.ActionManagePermissions)
	if !allowed {
		return nil, errors.Forbidden("list permissions")
	}

	perms, err := uc.permRepo.GetByResource(ctx, resourceUID, resType)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	resp := make([]*PermissionResponse, len(perms))
	for i, p := range perms {
		resp[i] = toPermissionResponse(p)
	}
	return &ListPermissionsResponse{Permissions: resp, Total: int64(len(resp))}, nil
}

func (uc *useCaseImpl) Check(
	ctx context.Context,
	req *CheckPermissionRequest,
) (*CheckPermissionResponse, error) {
	userUID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.BadRequest("invalid user ID")
	}
	resourceUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return nil, errors.BadRequest("invalid resource ID")
	}

	allowed, err := uc.permRepo.HasPermission(
		ctx,
		userUID,
		resourceUID,
		entity.ResourceType(req.ResourceType),
		entity.PermissionAction(req.Action),
	)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	return &CheckPermissionResponse{Allowed: allowed}, nil
}

func (uc *useCaseImpl) GrantBulk(
	ctx context.Context,
	grantedByID string,
	req *GrantBulkRequest,
) ([]*PermissionResponse, error) {
	granterUID, err := uuid.Parse(grantedByID)
	if err != nil {
		return nil, errors.BadRequest("invalid granter ID")
	}
	resourceUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return nil, errors.BadRequest("invalid resource ID")
	}

	resType := entity.ResourceType(req.ResourceType)
	allowed, err := uc.permRepo.HasPermission(ctx, granterUID, resourceUID, resType, entity.ActionManagePermissions)
	if err != nil {
		return nil, errors.InternalServer(err)
	}
	if !allowed {
		return nil, errors.Forbidden("manage permissions")
	}

	now := time.Now()
	perms := make([]*entity.Permission, 0, len(req.Permissions))
	for _, entry := range req.Permissions {
		targetUID, err := uuid.Parse(entry.UserID)
		if err != nil {
			return nil, errors.BadRequest("invalid user ID in bulk request")
		}
		perms = append(perms, &entity.Permission{
			ID:           uuid.New(),
			ResourceID:   resourceUID,
			ResourceType: resType,
			UserID:       targetUID,
			Action:       entity.PermissionAction(entry.Action),
			GrantedByID:  granterUID,
			ExpiresAt:    entry.ExpiresAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	if err := uc.permRepo.GrantBulk(ctx, perms); err != nil {
		return nil, errors.InternalServer(err)
	}

	resp := make([]*PermissionResponse, len(perms))
	for i, p := range perms {
		resp[i] = toPermissionResponse(p)
	}
	return resp, nil
}

func (uc *useCaseImpl) logAudit(
	ctx context.Context,
	userID, resourceID uuid.UUID,
	resourceType, action, status string,
	details map[string]interface{},
) {
	if uc.auditRepo == nil {
		return
	}
	rid := resourceID
	rtype := resourceType
	log := &entity.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       entity.AuditAction(action),
		ResourceID:   &rid,
		ResourceType: &rtype,
		Details:      details,
		Status:       status,
		CreatedAt:    time.Now(),
	}
	if err := uc.auditRepo.Create(ctx, log); err != nil {
		logger.Warn("failed to write audit log", zap.Error(err))
	}
}

func toPermissionResponse(p *entity.Permission) *PermissionResponse {
	return &PermissionResponse{
		ID:           p.ID.String(),
		UserID:       p.UserID.String(),
		ResourceID:   p.ResourceID.String(),
		ResourceType: string(p.ResourceType),
		Action:       string(p.Action),
		GrantedBy:    p.GrantedByID.String(),
		ExpiresAt:    p.ExpiresAt,
		CreatedAt:    p.CreatedAt,
	}
}
