package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"file-management-service/internal/delivery/http/middleware"
	notifinfra "file-management-service/internal/infrastructure/notification"
	"file-management-service/internal/usecase/notification"
	"file-management-service/pkg/logger"
	"file-management-service/pkg/response"
)

type NotificationHandler struct {
	notifUC   notification.UseCase
	publisher *notifinfra.Publisher
}

func NewNotificationHandler(notifUC notification.UseCase, publisher *notifinfra.Publisher) *NotificationHandler {
	return &NotificationHandler{notifUC: notifUC, publisher: publisher}
}

// List godoc
// @Summary      List notifications
// @Description  Get paginated list of notifications for the current user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int   false  "Page" default(1)
// @Param        page_size  query  int   false  "Page size" default(20)
// @Success      200  {object}  response.Response
// @Router       /notifications [get]
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := h.notifUC.List(c.Context(), userID, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "notifications retrieved", result)
}

// MarkAllAsRead godoc
// @Summary      Mark all notifications as read
// @Description  Mark all notifications for the current user as read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /notifications/read [post]
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if err := h.notifUC.MarkAllAsRead(c.Context(), userID); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "all notifications marked as read", nil)
}

// MarkAsRead godoc
// @Summary      Mark notification as read
// @Description  Mark a specific notification as read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Notification ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /notifications/{id}/read [patch]
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	notifID := c.Params("id")
	userID := middleware.GetUserID(c)

	if err := h.notifUC.MarkAsRead(c.Context(), userID, notifID); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "notification marked as read", nil)
}

// Delete godoc
// @Summary      Delete notification
// @Description  Delete a specific notification
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Notification ID (UUID)"
// @Success      204
// @Router       /notifications/{id} [delete]
func (h *NotificationHandler) Delete(c *fiber.Ctx) error {
	notifID := c.Params("id")
	userID := middleware.GetUserID(c)

	if err := h.notifUC.Delete(c.Context(), userID, notifID); err != nil {
		return handleError(c, err)
	}
	return response.NoContent(c)
}

// GetUnreadCount godoc
// @Summary      Get unread notification count
// @Description  Get the count of unread notifications
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /notifications/count [get]
func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	result, err := h.notifUC.GetUnreadCount(c.Context(), userID)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "unread count", result)
}

// Stream godoc
// @Summary      SSE notification stream
// @Description  Server-Sent Events stream for real-time notifications. Connect and listen for 'notification' events.
// @Tags         notifications
// @Produce      text/event-stream
// @Security     BearerAuth
// @Success      200
// @Router       /notifications/stream [get]
func (h *NotificationHandler) Stream(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // disable nginx buffering

	ctx := c.Context()
	pubsub := h.publisher.Subscribe(c.Context(), userID)
	defer pubsub.Close()

	ch := pubsub.Channel()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		connected := map[string]interface{}{
			"type":      "connected",
			"user_id":   userID,
			"timestamp": time.Now().UTC(),
		}
		writeSSEEvent(w, "connected", connected)

		// Heartbeat ticker to keep connection alive through proxies.
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var payload map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
					logger.Warn("SSE: failed to unmarshal notification payload",
						zap.String("user_id", userID), zap.Error(err))
					continue
				}
				writeSSEEvent(w, "notification", payload)

			case <-ticker.C:
				writeSSEEvent(w, "heartbeat", map[string]interface{}{
					"timestamp": time.Now().UTC(),
				})

			case <-ctx.Done():
				return
			}

			if err := w.Flush(); err != nil {
				logger.Debug("SSE: client disconnected", zap.String("user_id", userID))
				return
			}
		}
	})

	return nil
}

func writeSSEEvent(w *bufio.Writer, event string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
}
