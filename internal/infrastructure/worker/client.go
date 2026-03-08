package worker

import (
	"fmt"

	"github.com/hibiken/asynq"
)

type Client struct {
	client *asynq.Client
}

func NewClient(redisOpt asynq.RedisClientOpt) *Client {
	return &Client{client: asynq.NewClient(redisOpt)}
}

func (c *Client) EnqueueFileProcessing(payload *FileProcessingPayload, opts ...asynq.Option) error {
	task, err := NewFileProcessingTask(payload)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(task, opts...)
	return err
}

func (c *Client) EnqueueNotification(payload *SendNotificationPayload, opts ...asynq.Option) error {
	task, err := NewSendNotificationTask(payload)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(task, opts...)
	return err
}

// Enqueue submits any pre-built asynq.Task to the queue.
func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) error {
	if _, err := c.client.Enqueue(task, opts...); err != nil {
		return fmt.Errorf("enqueueing task %q: %w", task.Type(), err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
