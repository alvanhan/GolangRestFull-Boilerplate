package router

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "file-management-service/docs"

	"file-management-service/config"
	"file-management-service/internal/delivery/http/handler"
	"file-management-service/internal/delivery/http/middleware"
)

// Router wires all HTTP routes and middleware for the Fiber application.
type Router struct {
	cfg           *config.Config
	authHandler   *handler.AuthHandler
	fileHandler   *handler.FileHandler
	folderHandler *handler.FolderHandler
	permHandler   *handler.PermissionHandler
	notifHandler  *handler.NotificationHandler
	auditHandler  *handler.AuditHandler
	adminHandler  *handler.AdminHandler
	authMW        *middleware.AuthMiddleware
	rbacMW        *middleware.RBACMiddleware
	redisClient   *redis.Client
}

// NewRouter creates a Router with all dependencies injected.
func NewRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	fileHandler *handler.FileHandler,
	folderHandler *handler.FolderHandler,
	permHandler *handler.PermissionHandler,
	notifHandler *handler.NotificationHandler,
	auditHandler *handler.AuditHandler,
	adminHandler *handler.AdminHandler,
	authMW *middleware.AuthMiddleware,
	rbacMW *middleware.RBACMiddleware,
	redisClient *redis.Client,
) *Router {
	return &Router{
		cfg:           cfg,
		authHandler:   authHandler,
		fileHandler:   fileHandler,
		folderHandler: folderHandler,
		permHandler:   permHandler,
		notifHandler:  notifHandler,
		auditHandler:  auditHandler,
		adminHandler:  adminHandler,
		authMW:        authMW,
		rbacMW:        rbacMW,
		redisClient:   redisClient,
	}
}

// Setup creates and configures the Fiber application.
func (r *Router) Setup() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:           r.cfg.App.Name,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		BodyLimit:         int(r.cfg.Upload.MaxSize),
		ErrorHandler:      errorHandler,
		EnablePrintRoutes: !r.cfg.App.IsProduction(),
	})

	r.registerGlobalMiddleware(app)
	r.registerRoutes(app)
	return app
}

func (r *Router) registerGlobalMiddleware(app *fiber.App) {
	app.Use(recover.New(recover.Config{EnableStackTrace: !r.cfg.App.IsProduction()}))
	app.Use(requestid.New())
	app.Use(helmet.New())

	allowedOrigins := "*"
	if len(r.cfg.CORS.AllowedOrigins) > 0 {
		allowedOrigins = strings.Join(r.cfg.CORS.AllowedOrigins, ",")
	}
	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	if len(r.cfg.CORS.AllowedMethods) > 0 {
		allowedMethods = strings.Join(r.cfg.CORS.AllowedMethods, ",")
	}
	allowedHeaders := "Origin,Content-Type,Accept,Authorization,X-Request-ID"
	if len(r.cfg.CORS.AllowedHeaders) > 0 {
		allowedHeaders = strings.Join(r.cfg.CORS.AllowedHeaders, ",")
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     allowedMethods,
		AllowHeaders:     allowedHeaders,
		AllowCredentials: true,
	}))

	app.Use(middleware.RequestLogger())
	app.Use(middleware.RateLimit(&r.cfg.RateLimit, r.redisClient))
}

func (r *Router) registerRoutes(app *fiber.App) {
	auth := r.authMW.Authenticate()
	requireAdmin := r.rbacMW.RequireAdmin()

	// Swagger UI
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"service":   "file-management-service",
			"timestamp": time.Now().UTC(),
		})
	})

	v1 := app.Group("/api/v1")

	// ─── Auth ──────────────────────────────────────────────────────────────────
	authGroup := v1.Group("/auth")
	authGroup.Post("/register", r.authHandler.Register)
	authGroup.Post("/login", r.authHandler.Login)
	authGroup.Post("/refresh", r.authHandler.RefreshToken)
	authGroup.Post("/logout", auth, r.authHandler.Logout)
	authGroup.Post("/logout-all", auth, r.authHandler.LogoutAll)
	authGroup.Post("/change-password", auth, r.authHandler.ChangePassword)
	authGroup.Get("/me", auth, r.authHandler.GetProfile)
	authGroup.Put("/me", auth, r.authHandler.UpdateProfile)

	// ─── Public share links ────────────────────────────────────────────────────
	v1.Get("/share/:token", r.fileHandler.DownloadByShareToken)

	// ─── Files ─────────────────────────────────────────────────────────────────
	files := v1.Group("/files", auth)
	files.Get("/search", r.fileHandler.Search)
	files.Post("/upload", r.fileHandler.Upload)
	files.Post("/upload/init", r.fileHandler.InitChunkUpload)
	files.Post("/upload/chunk", r.fileHandler.UploadChunk)
	files.Post("/upload/complete", r.fileHandler.CompleteChunkUpload)
	files.Get("", r.fileHandler.List)
	files.Get("/:id", r.fileHandler.GetByID)
	files.Get("/:id/download", r.fileHandler.Download)
	files.Get("/:id/presigned", r.fileHandler.GetPresignedURL)
	files.Patch("/:id/move", r.fileHandler.Move)
	files.Patch("/:id/rename", r.fileHandler.Rename)
	files.Post("/:id/copy", r.fileHandler.Copy)
	files.Delete("/:id", r.fileHandler.Delete)
	files.Post("/:id/share", r.fileHandler.Share)
	files.Get("/:id/versions", r.fileHandler.GetVersions)
	files.Post("/:id/versions/:ver/restore", r.fileHandler.RestoreVersion)

	// ─── Folders ───────────────────────────────────────────────────────────────
	folders := v1.Group("/folders", auth)
	folders.Post("", r.folderHandler.Create)
	folders.Get("", r.folderHandler.List)
	folders.Get("/:id", r.folderHandler.GetByID)
	folders.Put("/:id", r.folderHandler.Update)
	folders.Delete("/:id", r.folderHandler.Delete)
	folders.Patch("/:id/move", r.folderHandler.Move)
	folders.Get("/:id/tree", r.folderHandler.GetTree)
	folders.Get("/:id/breadcrumb", r.folderHandler.GetBreadcrumb)
	folders.Post("/:id/share", r.folderHandler.Share)

	// ─── Permissions ───────────────────────────────────────────────────────────
	perms := v1.Group("/permissions", auth)
	perms.Post("", r.permHandler.Grant)
	perms.Post("/bulk", r.permHandler.GrantBulk)
	perms.Get("/resource", r.permHandler.List)
	perms.Post("/check", r.permHandler.Check)
	perms.Delete("/:id", r.permHandler.Revoke)

	// ─── Notifications ─────────────────────────────────────────────────────────
	notifs := v1.Group("/notifications", auth)
	notifs.Get("", r.notifHandler.List)
	notifs.Get("/count", r.notifHandler.GetUnreadCount)
	notifs.Get("/stream", r.notifHandler.Stream)
	notifs.Post("/read", r.notifHandler.MarkAllAsRead)
	notifs.Patch("/:id/read", r.notifHandler.MarkAsRead)
	notifs.Delete("/:id", r.notifHandler.Delete)

	// ─── Audit logs ────────────────────────────────────────────────────────────
	auditGroup := v1.Group("/audit-logs", auth, requireAdmin)
	auditGroup.Get("", r.auditHandler.List)
	auditGroup.Get("/export", r.auditHandler.Export)
	auditGroup.Get("/:id", r.auditHandler.GetByID)

	// ─── Admin ─────────────────────────────────────────────────────────────────
	adminGroup := v1.Group("/admin", auth, requireAdmin)
	adminGroup.Get("/stats", r.adminHandler.GetStats)
	adminGroup.Get("/users", r.adminHandler.ListUsers)
	adminGroup.Post("/users", r.adminHandler.CreateUser)
	adminGroup.Get("/users/:id", r.adminHandler.GetUser)
	adminGroup.Put("/users/:id", r.adminHandler.UpdateUser)
	adminGroup.Delete("/users/:id", r.adminHandler.DeleteUser)
	adminGroup.Post("/users/:id/ban", r.adminHandler.BanUser)
}

// errorHandler is the global Fiber error handler.
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"message": msg,
		"error": fiber.Map{
			"code":    code,
			"message": msg,
		},
	})
}
