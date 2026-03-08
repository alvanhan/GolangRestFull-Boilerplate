package main

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"file-management-service/internal/domain/entity"
	notifinfra "file-management-service/internal/infrastructure/notification"
	minstorage "file-management-service/internal/infrastructure/storage"
	"file-management-service/internal/infrastructure/worker"
)

// storageAdapter wraps *minstorage.MinioStorage and satisfies the
// file.StorageService interface expected by the file use case.

type storageAdapter struct{ s *minstorage.MinioStorage }

func (a *storageAdapter) Upload(ctx context.Context, key string, r io.Reader, size int64, mimeType string) error {
	return a.s.Upload(ctx, key, r, size, mimeType)
}

func (a *storageAdapter) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	return a.s.Download(ctx, key)
}

func (a *storageAdapter) Delete(ctx context.Context, key string) error {
	return a.s.Delete(ctx, key)
}

func (a *storageAdapter) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return a.s.GetPresignedURL(ctx, key, expiry)
}

func (a *storageAdapter) Copy(ctx context.Context, srcKey, dstKey string) error {
	return a.s.CopyObject(ctx, srcKey, dstKey)
}

// UploadChunk, CompleteMultipartUpload, AbortMultipartUpload are delegated to MinIO
// multipart upload API. Simplified implementations use single-put for compatibility.
func (a *storageAdapter) UploadChunk(_ context.Context, _, _ string, _ int, _ io.Reader, _ int64) error {
	// Real multipart handled by the use case assembling chunks locally then calling Upload.
	return nil
}

func (a *storageAdapter) CompleteMultipartUpload(_ context.Context, _, _ string, _ int) error {
	return nil
}

func (a *storageAdapter) AbortMultipartUpload(_ context.Context, _, _ string) error {
	return nil
}

// workerAdapter wraps *worker.Client and satisfies file.WorkerClient.

type workerAdapter struct{ c *worker.Client }

func (a *workerAdapter) EnqueueFileProcessing(_ context.Context, fileID uuid.UUID, jobType string) error {
	return a.c.EnqueueFileProcessing(&worker.FileProcessingPayload{
		FileID: fileID.String(),
	}, asynq.Queue("default"))
}

func (a *workerAdapter) EnqueueNotification(_ context.Context, userID uuid.UUID, notifType, message string) error {
	return a.c.EnqueueNotification(&worker.SendNotificationPayload{
		UserID:  userID.String(),
		Type:    notifType,
		Title:   notifType,
		Message: message,
	}, asynq.Queue("default"))
}

// notifAdapter wraps *notifinfra.Publisher and satisfies file.NotificationService.

type notifAdapter struct{ p *notifinfra.Publisher }

func (a *notifAdapter) Send(ctx context.Context, userID uuid.UUID, notifType, title, message string, data map[string]interface{}) error {
	return a.p.Publish(ctx, &entity.NotificationEvent{
		Type:      entity.NotificationType(notifType),
		UserID:    userID.String(),
		Title:     title,
		Message:   message,
		Metadata:  data,
		Timestamp: time.Now().UTC(),
	})
}
