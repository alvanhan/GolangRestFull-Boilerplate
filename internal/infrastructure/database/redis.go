package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"file-management-service/config"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(cfg *config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.Client.Close()
}

func (r *RedisClient) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.Client.FlushDB(ctx).Err()
}
