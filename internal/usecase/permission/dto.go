package permission

import "time"

// GrantPermissionRequest grants a permission to a user on a resource.
type GrantPermissionRequest struct {
	UserID       string     `json:"user_id"        validate:"required,uuid"`
	ResourceID   string     `json:"resource_id"    validate:"required,uuid"`
	ResourceType string     `json:"resource_type"  validate:"required,oneof=file folder"`
	Action       string     `json:"action"         validate:"required,oneof=read write delete download share upload manage_permissions"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

// RevokePermissionRequest revokes a specific permission grant by its ID.
type RevokePermissionRequest struct {
	PermissionID string `json:"permission_id" validate:"required,uuid"`
}

// GrantBulkRequest grants multiple permissions at once on the same resource.
type GrantBulkRequest struct {
	ResourceID   string                     `json:"resource_id"   validate:"required,uuid"`
	ResourceType string                     `json:"resource_type" validate:"required,oneof=file folder"`
	Permissions  []BulkPermissionEntry      `json:"permissions"   validate:"required,min=1,dive"`
}

// BulkPermissionEntry is a single entry in a bulk-grant request.
type BulkPermissionEntry struct {
	UserID    string     `json:"user_id"  validate:"required,uuid"`
	Action    string     `json:"action"   validate:"required,oneof=read write delete download share upload manage_permissions"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// PermissionResponse is the public representation of a permission grant.
type PermissionResponse struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	ResourceID   string     `json:"resource_id"`
	ResourceType string     `json:"resource_type"`
	Action       string     `json:"action"`
	GrantedBy    string     `json:"granted_by"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ListPermissionsResponse wraps a paginated list of permissions.
type ListPermissionsResponse struct {
	Permissions []*PermissionResponse `json:"permissions"`
	Total       int64                 `json:"total"`
}

// CheckPermissionRequest queries whether a user has a specific permission.
type CheckPermissionRequest struct {
	UserID       string `json:"user_id"        validate:"required,uuid"`
	ResourceID   string `json:"resource_id"    validate:"required,uuid"`
	ResourceType string `json:"resource_type"  validate:"required,oneof=file folder"`
	Action       string `json:"action"         validate:"required"`
}

// CheckPermissionResponse holds the result of a permission check.
type CheckPermissionResponse struct {
	Allowed bool `json:"allowed"`
}
