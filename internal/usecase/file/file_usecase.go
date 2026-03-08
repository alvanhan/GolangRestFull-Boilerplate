package file

import (
	"context"
	"io"
	"time"
)

// UseCase is the file-management business-logic contract.
type UseCase interface {
	// Upload stores a complete file in one shot.
	Upload(ctx context.Context, ownerID string, req *UploadFileRequest,
		filename string, fileReader io.Reader, fileSize int64) (*FileResponse, error)

	// InitChunkUpload starts a multipart upload session and returns the session ID.
	InitChunkUpload(ctx context.Context, ownerID string,
		req *InitChunkUploadRequest) (*ChunkUploadInitResponse, error)

	// UploadChunk stores a single chunk belonging to an open upload session.
	UploadChunk(ctx context.Context, ownerID, uploadID string,
		chunkIndex int, chunkReader io.Reader, chunkSize int64, checksum string) error

	// CompleteChunkUpload assembles all chunks and creates the final file record.
	CompleteChunkUpload(ctx context.Context, ownerID, uploadID string) (*FileResponse, error)

	// Download returns a readable stream and the file metadata for a file.
	Download(ctx context.Context, userID, fileID string) (io.ReadCloser, *FileResponse, error)

	// GetPresignedURL returns a time-limited URL for direct object download.
	GetPresignedURL(ctx context.Context, userID, fileID string, expiry time.Duration) (string, error)

	// Delete permanently removes a file the caller owns or has delete permission on.
	Delete(ctx context.Context, userID, fileID string) error

	// Move relocates a file to a different folder (nil = root).
	Move(ctx context.Context, userID, fileID string, req *MoveFileRequest) (*FileResponse, error)

	// Rename changes the display name of a file.
	Rename(ctx context.Context, userID, fileID string, req *RenameFileRequest) (*FileResponse, error)

	// Copy duplicates a file into the given folder (nil = root).
	Copy(ctx context.Context, userID, fileID string, targetFolderID *string) (*FileResponse, error)

	// GetByID returns a file's metadata if the caller has at least read access.
	GetByID(ctx context.Context, userID, fileID string) (*FileResponse, error)

	// List returns files in a folder (nil = root) with optional filtering.
	List(ctx context.Context, userID string, folderID *string,
		filter *FileListFilter) ([]*FileResponse, int64, error)

	// Search returns files matching a free-text query.
	Search(ctx context.Context, userID, query string) ([]*FileResponse, error)

	// Share creates a share link for a file.
	Share(ctx context.Context, userID, fileID string,
		req *ShareFileRequest) (*ShareLinkResponse, error)

	// GetVersions lists all historical versions of a file.
	GetVersions(ctx context.Context, userID, fileID string) ([]*FileVersionResponse, error)

	// RestoreVersion rolls back a file to the specified version number.
	RestoreVersion(ctx context.Context, userID, fileID string, version int) (*FileResponse, error)

	// DownloadByShareToken streams a file identified by a public share token.
	DownloadByShareToken(ctx context.Context, token string) (io.ReadCloser, *FileResponse, error)
}
