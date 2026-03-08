package file

import "time"

// UploadFileRequest carries metadata for a simple (non-chunked) upload.
type UploadFileRequest struct {
	FolderID    *string  `form:"folder_id"`
	Description *string  `form:"description"`
	Tags        []string `form:"tags"`
}

// InitChunkUploadRequest starts a chunked upload session.
type InitChunkUploadRequest struct {
	FileName    string  `json:"file_name"    validate:"required"`
	FileSize    int64   `json:"file_size"    validate:"required,min=1"`
	MimeType    string  `json:"mime_type"    validate:"required"`
	TotalChunks int     `json:"total_chunks" validate:"required,min=1"`
	FolderID    *string `json:"folder_id"`
}

// UploadChunkRequest carries metadata for a single chunk.
type UploadChunkRequest struct {
	UploadID   string `form:"upload_id"    validate:"required"`
	ChunkIndex int    `form:"chunk_index"  validate:"min=0"`
	Checksum   string `form:"checksum"     validate:"required"`
}

// MoveFileRequest specifies the destination folder for a move operation.
type MoveFileRequest struct {
	// TargetFolderID nil means move to the root.
	TargetFolderID *string `json:"target_folder_id"`
}

// RenameFileRequest carries the new name for a file.
type RenameFileRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// ShareFileRequest creates a share link for a file.
type ShareFileRequest struct {
	ExpiresAt *time.Time `json:"expires_at"`
	MaxUses   *int       `json:"max_uses"`
	Password  *string    `json:"password"`
	Action    string     `json:"action" validate:"required,oneof=read download write"`
}

// FileListFilter holds optional search/filter parameters for listing files.
type FileListFilter struct {
	MimeType  *string
	Status    *string
	Tags      []string
	Search    string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

// FileResponse is the public representation of a file.
type FileResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	OriginalName  string     `json:"original_name"`
	Extension     string     `json:"extension"`
	MimeType      string     `json:"mime_type"`
	Size          int64      `json:"size"`
	SizeFormatted string     `json:"size_formatted"`
	FolderID      *string    `json:"folder_id,omitempty"`
	OwnerID       string     `json:"owner_id"`
	Version       int        `json:"version"`
	Status        string     `json:"status"`
	IsPublic      bool       `json:"is_public"`
	DownloadCount int64      `json:"download_count"`
	Tags          []string   `json:"tags"`
	Description   *string    `json:"description,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// FileVersionResponse represents a single historical version of a file.
type FileVersionResponse struct {
	ID         string    `json:"id"`
	FileID     string    `json:"file_id"`
	Version    int       `json:"version"`
	Size       int64     `json:"size"`
	Checksum   string    `json:"checksum"`
	CreatedBy  string    `json:"created_by"`
	ChangeNote *string   `json:"change_note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ShareLinkResponse is the public representation of a generated share link.
type ShareLinkResponse struct {
	Token     string     `json:"token"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	Action    string     `json:"action"`
	CreatedAt time.Time  `json:"created_at"`
}

// ChunkUploadInitResponse is returned when a chunked upload session is created.
type ChunkUploadInitResponse struct {
	UploadID    string    `json:"upload_id"`
	ChunkSize   int64     `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	ExpiresAt   time.Time `json:"expires_at"`
}
