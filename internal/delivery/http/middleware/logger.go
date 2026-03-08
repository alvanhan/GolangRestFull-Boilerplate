package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"file-management-service/pkg/logger"
)

const headerRequestID = "X-Request-ID"

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(headerRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
			c.Set(headerRequestID, reqID)
		}

		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()

		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get(fiber.HeaderUserAgent)),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		}

		if userID := GetUserID(c); userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		if status >= 500 {
			logger.Error("request completed with server error", fields...)
		} else if status >= 400 {
			logger.Warn("request completed with client error", fields...)
		} else {
			logger.Info("request completed", fields...)
		}

		return err
	}
}
