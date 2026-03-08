// @title           File Management Service API
// @version         1.0
// @description     Enterprise-grade file management service with RBAC, chunked uploads, and real-time notifications.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@filemanagement.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"file-management-service/config"
	"file-management-service/internal/delivery/http/handler"
	"file-management-service/internal/delivery/http/middleware"
	"file-management-service/internal/delivery/http/router"
	"file-management-service/internal/infrastructure/cache"
	"file-management-service/internal/infrastructure/database"
	"file-management-service/internal/infrastructure/notification"
	"file-management-service/internal/infrastructure/repository"
	"file-management-service/internal/infrastructure/storage"
	"file-management-service/internal/infrastructure/worker"
	"file-management-service/internal/usecase/auth"
	fileuc "file-management-service/internal/usecase/file"
	"file-management-service/internal/usecase/folder"
	notifuc "file-management-service/internal/usecase/notification"
	"file-management-service/internal/usecase/permission"
	"file-management-service/pkg/jwt"
	"file-management-service/pkg/logger"
	"file-management-service/pkg/validator"
)

// Version and BuildTime are injected at link time via -ldflags in the Makefile.
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 1. Load and validate configuration.
	// config.Load() panics with a descriptive message on any invalid or missing
	// required environment variable, so no explicit error check is needed here.
	cfg := config.Load()

	if err := logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting file management service",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	db, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}

	// 4. Run GORM structural auto-migrations for all registered entity models.
	// This is a safety net for development; production should use SQL migration
	// files (see migrations/ directory) run by golang-migrate.
	if err := db.AutoMigrate(); err != nil {
		logger.Fatal("Database auto-migration failed", zap.Error(err))
	}

	// 5. Connect to Redis (used for caching, pub/sub, and Asynq job queues).
	redisClient, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	minioStorage, err := storage.NewMinioStorage(&cfg.MinIO)
	if err != nil {
		logger.Fatal("Failed to initialise MinIO storage", zap.Error(err))
	}
	initCtx := context.Background()
	if err := minioStorage.EnsureBucket(initCtx); err != nil {
		logger.Fatal("Failed to ensure MinIO bucket exists",
			zap.Error(err),
			zap.String("bucket", cfg.MinIO.BucketName),
		)
	}
	logger.Info("MinIO storage ready", zap.String("bucket", cfg.MinIO.BucketName))

	_ = cache.NewRedisCache(redisClient.Client)

	// 8. Initialise the Redis pub/sub notification publisher.
	// Delivers real-time events to connected SSE clients.
	notifPublisher := notification.NewPublisher(redisClient.Client)

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.GetRedisAddr(),
		Password: cfg.Redis.Password,
	}
	workerClient := worker.NewClient(redisOpt)
	defer workerClient.Close()

	userRepo := repository.NewUserRepository(db.DB)
	fileRepo := repository.NewFileRepository(db.DB)
	folderRepo := repository.NewFolderRepository(db.DB)
	permRepo := repository.NewPermissionRepository(db.DB)
	auditRepo := repository.NewAuditRepository(db.DB)
	notifRepo := repository.NewNotificationRepository(db.DB)

	jwtService := jwt.NewJWTService(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	v := validator.New()

	authUC := auth.NewUseCase(userRepo, jwtService)
	fileUC := fileuc.NewUseCase(
		fileRepo, folderRepo, permRepo, userRepo, auditRepo,
		&storageAdapter{minioStorage}, &workerAdapter{workerClient}, &notifAdapter{notifPublisher}, &cfg.Upload,
	)
	folderUC := folder.NewUseCase(folderRepo, permRepo, auditRepo, notifPublisher)
	permUC := permission.NewUseCase(permRepo, fileRepo, folderRepo, auditRepo, notifPublisher)
	notifUC := notifuc.NewUseCase(notifRepo)

	authHandler := handler.NewAuthHandler(authUC, v)
	fileHandler := handler.NewFileHandler(fileUC, v, &cfg.Upload)
	folderHandler := handler.NewFolderHandler(folderUC, v)
	permHandler := handler.NewPermissionHandler(permUC, v)
	notifHandler := handler.NewNotificationHandler(notifUC, notifPublisher)
	auditHandler := handler.NewAuditHandler(auditRepo, v)
	adminHandler := handler.NewAdminHandler(userRepo, fileRepo, v)

	authMW := middleware.NewAuthMiddleware(jwtService)
	rbacMW := middleware.NewRBACMiddleware()

	r := router.NewRouter(
		cfg,
		authHandler, fileHandler, folderHandler,
		permHandler, notifHandler, auditHandler, adminHandler,
		authMW, rbacMW,
		redisClient.Client,
	)
	app := r.Setup()

	processor := worker.NewProcessor(&cfg.Worker, redisOpt)
	fileProcessingHandler := worker.NewFileProcessingHandler()
	notifWorkerHandler := worker.NewNotificationHandler(notifPublisher, notifRepo)
	processor.RegisterHandlers(fileProcessingHandler, notifWorkerHandler)
	go func() {
		logger.Info("Worker processor starting",
			zap.Int("concurrency", cfg.Worker.Concurrency),
			zap.String("queue_default", cfg.Worker.QueueDefault),
			zap.String("queue_critical", cfg.Worker.QueueCritical),
		)
		if err := processor.Start(); err != nil {
			logger.Error("Worker processor stopped with error", zap.Error(err))
		}
	}()
	defer processor.Stop()

	scheduler := worker.NewScheduler(workerClient)
	scheduler.RegisterJobs()
	scheduler.Start()
	defer scheduler.Stop()

	serverAddr := fmt.Sprintf(":%d", cfg.App.Port)
	go func() {
		logger.Info("HTTP server listening", zap.String("addr", serverAddr))
		if err := app.Listen(serverAddr); err != nil {
			logger.Error("HTTP server stopped with error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received, draining connections…",
		zap.String("signal", sig.String()),
	)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown encountered an error", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}
