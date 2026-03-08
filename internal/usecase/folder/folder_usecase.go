package folder

import "context"

// UseCase is the folder management business-logic contract.
type UseCase interface {
	// Create makes a new folder under an optional parent.
	Create(ctx context.Context, ownerID string, req *CreateFolderRequest) (*FolderResponse, error)

	// GetByID returns a folder the caller has at least read access to.
	GetByID(ctx context.Context, userID, folderID string) (*FolderResponse, error)

	// Update applies partial changes to a folder.
	Update(ctx context.Context, userID, folderID string, req *UpdateFolderRequest) (*FolderResponse, error)

	// Delete recursively removes a folder and all its contents.
	Delete(ctx context.Context, userID, folderID string) error

	// Move relocates a folder under a new parent (nil = root).
	Move(ctx context.Context, userID, folderID string, req *MoveFolderRequest) (*FolderResponse, error)

	// List returns the immediate children of a folder (nil = root).
	List(ctx context.Context, userID string, parentID *string) ([]*FolderResponse, error)

	// GetTree returns the full recursive subtree rooted at folderID (nil = entire tree).
	GetTree(ctx context.Context, userID string, folderID *string) (*FolderTreeResponse, error)

	// GetBreadcrumb returns the ancestor chain from root to folderID.
	GetBreadcrumb(ctx context.Context, userID, folderID string) ([]*BreadcrumbItem, error)

	// Share creates a share link for a folder.
	Share(ctx context.Context, userID, folderID string, req *ShareFolderRequest) (*ShareLinkResponse, error)
}
