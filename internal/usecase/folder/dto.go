package folder

import "time"

// CreateFolderRequest holds data for creating a new folder.
type CreateFolderRequest struct {
	Name        string  `json:"name"        validate:"required,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	ParentID    *string `json:"parent_id"`
	Color       *string `json:"color"       validate:"omitempty,max=20"`
}

// UpdateFolderRequest carries optional updates for an existing folder.
type UpdateFolderRequest struct {
	Name        *string `json:"name"        validate:"omitempty,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Color       *string `json:"color"       validate:"omitempty,max=20"`
}

// MoveFolderRequest specifies the destination parent for a move operation.
type MoveFolderRequest struct {
	// TargetParentID nil means move to the root.
	TargetParentID *string `json:"target_parent_id"`
}

// ShareFolderRequest creates a share link for a folder.
type ShareFolderRequest struct {
	ExpiresAt *time.Time `json:"expires_at"`
	MaxUses   *int       `json:"max_uses"`
	Password  *string    `json:"password"`
	Action    string     `json:"action" validate:"required,oneof=read download write"`
}

// FolderResponse is the public representation of a folder.
type FolderResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	ParentID     *string    `json:"parent_id,omitempty"`
	OwnerID      string     `json:"owner_id"`
	Path         string     `json:"path"`
	IsRoot       bool       `json:"is_root"`
	IsShared     bool       `json:"is_shared"`
	Color        *string    `json:"color,omitempty"`
	ChildrenCount int64     `json:"children_count"`
	FileCount    int64      `json:"file_count"`
	Size         int64      `json:"size"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FolderTreeResponse represents a folder with its recursive children.
type FolderTreeResponse struct {
	FolderResponse
	Children []*FolderTreeResponse `json:"children,omitempty"`
}

// BreadcrumbItem is a single element of a folder's navigation path.
type BreadcrumbItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// ShareLinkResponse mirrors the file use case's share response.
type ShareLinkResponse struct {
	Token     string     `json:"token"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	Action    string     `json:"action"`
	CreatedAt time.Time  `json:"created_at"`
}
