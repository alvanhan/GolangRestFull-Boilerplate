package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling value: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// Get retrieves a cached value and unmarshals it into dest. Returns an error if the key is not found.
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key %q not found", key)
		}
		return fmt.Errorf("getting key %q: %w", key, err)
	}
	return json.Unmarshal(data, dest)
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// SetNX sets the key only if it does not already exist (atomic). Returns true if the key was set.
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("marshaling value: %w", err)
	}
	return c.client.SetNX(ctx, key, data, ttl).Result()
}

func (c *RedisCache) SetUserSession(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	return c.Set(ctx, c.CacheKey("session", userID), data, ttl)
}

func (c *RedisCache) GetUserSession(ctx context.Context, userID string, dest interface{}) error {
	return c.Get(ctx, c.CacheKey("session", userID), dest)
}

func (c *RedisCache) DeleteUserSession(ctx context.Context, userID string) error {
	return c.Delete(ctx, c.CacheKey("session", userID))
}

// CacheKey joins the given parts with ":" to form a namespaced cache key.
func (c *RedisCache) CacheKey(parts ...string) string {
	return strings.Join(parts, ":")
}
