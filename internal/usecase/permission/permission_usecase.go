package permission

import "context"

// UseCase is the permission-management business-logic contract.
type UseCase interface {
	// Grant creates a single permission grant.
	Grant(ctx context.Context, grantedByID string, req *GrantPermissionRequest) (*PermissionResponse, error)

	// Revoke removes a single permission grant by its ID.
	Revoke(ctx context.Context, revokerID, permissionID string) error

	// List returns all permission grants on a resource, visible to the caller.
	List(ctx context.Context, callerID, resourceID, resourceType string) (*ListPermissionsResponse, error)

	// Check reports whether a user has the requested permission on a resource.
	Check(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error)

	// GrantBulk atomically grants multiple permissions on the same resource.
	GrantBulk(ctx context.Context, grantedByID string, req *GrantBulkRequest) ([]*PermissionResponse, error)
}
