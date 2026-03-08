package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"file-management-service/config"
	"file-management-service/internal/domain/repository"
	notifpkg "file-management-service/internal/infrastructure/notification"
	"file-management-service/pkg/logger"
)

type Processor struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

type FileProcessingHandler struct{}

type NotificationHandler struct {
	publisher *notifpkg.Publisher
	repo      repository.NotificationRepository
}

func NewFileProcessingHandler() *FileProcessingHandler {
	return &FileProcessingHandler{}
}

func NewNotificationHandler(publisher *notifpkg.Publisher, repo repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{publisher: publisher, repo: repo}
}

func NewProcessor(cfg *config.WorkerConfig, redisOpt asynq.RedisClientOpt) *Processor {
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues: map[string]int{
			cfg.QueueCritical: 6,
			cfg.QueueDefault:  3,
		},
	})

	return &Processor{
		server: srv,
		mux:    asynq.NewServeMux(),
	}
}

func (p *Processor) RegisterHandlers(fileHandler *FileProcessingHandler, notifHandler *NotificationHandler) {
	p.mux.HandleFunc(TypeFileProcessing, fileHandler.ProcessTask)
	p.mux.HandleFunc(TypeFileCleanup, handleFileCleanup)
	p.mux.HandleFunc(TypeSendNotification, notifHandler.ProcessTask)
	p.mux.HandleFunc(TypeAuditCleanup, handleAuditCleanup)
	p.mux.HandleFunc(TypeStorageReport, handleStorageReport)
	p.mux.HandleFunc(TypeExpiredLinkCleanup, handleExpiredLinkCleanup)
	p.mux.HandleFunc(TypeChunkCleanup, handleChunkCleanup)
}

func (p *Processor) Start() error {
	return p.server.Start(p.mux)
}

func (p *Processor) Stop() {
	p.server.Shutdown()
}

func (h *FileProcessingHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload FileProcessingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshaling file processing payload: %w", err)
	}

	logger.Info("processing file",
		zap.String("file_id", payload.FileID),
		zap.String("mime_type", payload.MimeType),
		zap.String("owner_id", payload.OwnerID),
	)
	// TODO: implement actual processing (thumbnail generation, virus scan, metadata extraction, etc.)
	return nil
}

func (h *NotificationHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload SendNotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshaling send notification payload: %w", err)
	}

	logger.Info("sending notification",
		zap.String("user_id", payload.UserID),
		zap.String("type", payload.Type),
		zap.String("title", payload.Title),
	)
	// TODO: persist notification record and publish via pub/sub
	return nil
}

func handleFileCleanup(ctx context.Context, t *asynq.Task) error {
	var payload FileCleanupPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshaling file cleanup payload: %w", err)
	}
	logger.Info("cleaning up file",
		zap.String("file_id", payload.FileID),
		zap.String("storage_key", payload.StorageKey),
	)
	return nil
}

func handleAuditCleanup(_ context.Context, _ *asynq.Task) error {
	logger.Info("running audit log cleanup")
	return nil
}

func handleStorageReport(_ context.Context, _ *asynq.Task) error {
	logger.Info("generating storage report")
	return nil
}

func handleExpiredLinkCleanup(_ context.Context, _ *asynq.Task) error {
	logger.Info("cleaning up expired share links")
	return nil
}

func handleChunkCleanup(_ context.Context, _ *asynq.Task) error {
	logger.Info("cleaning up orphaned upload chunks")
	return nil
}
