package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeFileProcessing     = "file:processing"
	TypeFileCleanup        = "file:cleanup"
	TypeSendNotification   = "notification:send"
	TypeAuditCleanup       = "audit:cleanup"
	TypeStorageReport      = "storage:report"
	TypeExpiredLinkCleanup = "sharelink:cleanup"
	TypeChunkCleanup       = "chunk:cleanup"
)

type FileProcessingPayload struct {
	FileID     string `json:"file_id"`
	OwnerID    string `json:"owner_id"`
	StorageKey string `json:"storage_key"`
	MimeType   string `json:"mime_type"`
}

type FileCleanupPayload struct {
	FileID     string `json:"file_id"`
	StorageKey string `json:"storage_key"`
}

type SendNotificationPayload struct {
	UserID       string `json:"user_id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
}

type AuditCleanupPayload struct{}

type StorageReportPayload struct{}

type ExpiredLinkCleanupPayload struct{}

type ChunkCleanupPayload struct{}

func NewFileProcessingTask(payload *FileProcessingPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling file processing payload: %w", err)
	}
	return asynq.NewTask(TypeFileProcessing, data), nil
}

func NewFileCleanupTask(payload *FileCleanupPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling file cleanup payload: %w", err)
	}
	return asynq.NewTask(TypeFileCleanup, data), nil
}

func NewSendNotificationTask(payload *SendNotificationPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling send notification payload: %w", err)
	}
	return asynq.NewTask(TypeSendNotification, data), nil
}

func NewAuditCleanupTask() (*asynq.Task, error) {
	data, err := json.Marshal(AuditCleanupPayload{})
	if err != nil {
		return nil, fmt.Errorf("marshaling audit cleanup payload: %w", err)
	}
	return asynq.NewTask(TypeAuditCleanup, data), nil
}

func NewStorageReportTask() (*asynq.Task, error) {
	data, err := json.Marshal(StorageReportPayload{})
	if err != nil {
		return nil, fmt.Errorf("marshaling storage report payload: %w", err)
	}
	return asynq.NewTask(TypeStorageReport, data), nil
}

func NewExpiredLinkCleanupTask() (*asynq.Task, error) {
	data, err := json.Marshal(ExpiredLinkCleanupPayload{})
	if err != nil {
		return nil, fmt.Errorf("marshaling expired link cleanup payload: %w", err)
	}
	return asynq.NewTask(TypeExpiredLinkCleanup, data), nil
}

func NewChunkCleanupTask() (*asynq.Task, error) {
	data, err := json.Marshal(ChunkCleanupPayload{})
	if err != nil {
		return nil, fmt.Errorf("marshaling chunk cleanup payload: %w", err)
	}
	return asynq.NewTask(TypeChunkCleanup, data), nil
}
