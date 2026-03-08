package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"file-management-service/internal/domain/entity"
	"file-management-service/pkg/logger"
)

// notificationChannel is the per-user Redis pub/sub channel template.
const notificationChannel = "notifications:%s"

type Publisher struct {
	client *redis.Client
}

func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{client: client}
}

// Publish marshals the event and publishes it to the user's notification channel.
func (p *Publisher) Publish(ctx context.Context, event *entity.NotificationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling notification event: %w", err)
	}

	channel := p.UserChannel(event.UserID)
	if err := p.client.Publish(ctx, channel, data).Err(); err != nil {
		logger.Error("failed to publish notification",
			zap.String("user_id", event.UserID),
			zap.Error(err),
		)
		return fmt.Errorf("publishing notification: %w", err)
	}

	return nil
}

// Subscribe returns a PubSub handle for the given user's notification channel.
func (p *Publisher) Subscribe(ctx context.Context, userID string) *redis.PubSub {
	return p.client.Subscribe(ctx, p.UserChannel(userID))
}

// UserChannel returns the Redis channel name for the given user ID.
func (p *Publisher) UserChannel(userID string) string {
	return fmt.Sprintf(notificationChannel, userID)
}

// Send creates a NotificationEvent and publishes it to the user's channel.
// This method satisfies the NotificationService interface used by file/folder/permission use cases.
func (p *Publisher) Send(
	ctx context.Context,
	userID uuid.UUID,
	notifType, title, message string,
	data map[string]interface{},
) error {
	event := &entity.NotificationEvent{
		Type:      entity.NotificationType(notifType),
		UserID:    userID.String(),
		Title:     title,
		Message:   message,
		Metadata:  data,
		Timestamp: time.Now(),
	}
	return p.Publish(ctx, event)
}
