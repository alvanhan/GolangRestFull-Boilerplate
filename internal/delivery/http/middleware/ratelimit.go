package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"file-management-service/config"
)

// RateLimit returns a per-IP sliding-window rate limiter backed by Redis.
func RateLimit(cfg *config.RateLimitConfig, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		key := fmt.Sprintf("ratelimit:%s", ip)
		ctx := context.Background()

		pipe := redisClient.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, cfg.Expiry)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// On Redis error, allow the request through to avoid blocking legitimate users.
			return c.Next()
		}

		count := incr.Val()
		remaining := int64(cfg.Max) - count
		if remaining < 0 {
			remaining = 0
		}

		resetAt := time.Now().Add(cfg.Expiry)
		c.Set("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if count > int64(cfg.Max) {
			c.Set("Retry-After", strconv.FormatInt(int64(cfg.Expiry.Seconds()), 10))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "too many requests, please slow down",
				"error": fiber.Map{
					"code":    429,
					"message": "rate limit exceeded",
				},
			})
		}

		return c.Next()
	}
}
