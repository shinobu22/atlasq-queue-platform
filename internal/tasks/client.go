package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/logger"
)

type Client struct {
	asynqClient *asynq.Client
	redisClient *redis.Client
	config      *config.Config
}

type EnqueueOptions struct {
	Queue     string
	Delay     time.Duration
	MaxRetry  int
	Timeout   time.Duration
	UniqueTTL time.Duration
}

type EnqueueResult struct {
	TaskID      string    `json:"task_id"`
	Queue       string    `json:"queue"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func NewClient(cfg *config.Config) (*Client, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	asynqClient := asynq.NewClient(redisOpt)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &Client{
		asynqClient: asynqClient,
		redisClient: redisClient,
		config:      cfg,
	}, nil
}

func (c *Client) Close() error {
	if c.asynqClient != nil {
		c.asynqClient.Close()
	}
	if c.redisClient != nil {
		return c.redisClient.Close()
	}
	return nil
}

func (c *Client) EnqueueNow(ctx context.Context, taskType string, payload interface{}, opts *EnqueueOptions) (*EnqueueResult, error) {
	if opts == nil {
		opts = &EnqueueOptions{}
	}
	return c.enqueue(ctx, taskType, payload, opts)
}

func (c *Client) EnqueueDelayed(ctx context.Context, taskType string, payload interface{}, delay time.Duration, opts *EnqueueOptions) (*EnqueueResult, error) {
	if opts == nil {
		opts = &EnqueueOptions{}
	}
	opts.Delay = delay
	return c.enqueue(ctx, taskType, payload, opts)
}

func (c *Client) EnqueueUnique(ctx context.Context, taskType string, payload interface{}, uniqueTTL time.Duration, opts *EnqueueOptions) (*EnqueueResult, error) {
	if opts == nil {
		opts = &EnqueueOptions{}
	}
	opts.UniqueTTL = uniqueTTL
	return c.enqueue(ctx, taskType, payload, opts)
}

func (c *Client) enqueue(ctx context.Context, taskType string, payload interface{}, opts *EnqueueOptions) (*EnqueueResult, error) {
	log := logger.FromContext(ctx)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal task payload", zap.Error(err))
		return nil, err
	}

	if err := ValidateTaskPayload(taskType, payloadBytes); err != nil {
		log.Error("Task payload validation failed", 
			zap.String("task_type", taskType),
			zap.Error(err))
		return nil, err
	}

	task := asynq.NewTask(taskType, payloadBytes)

	queue := opts.Queue
	if queue == "" {
		queue = c.config.Asynq.ClientQueueDefault
	}

	maxRetry := opts.MaxRetry
	if maxRetry == 0 {
		maxRetry = c.config.Tasks.DefaultMaxRetry
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = c.config.Tasks.DefaultTimeout
	}

	asynqOpts := []asynq.Option{
		asynq.Queue(queue),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(timeout),
	}

	if opts.Delay > 0 {
		asynqOpts = append(asynqOpts, asynq.ProcessIn(opts.Delay))
	}

	if opts.UniqueTTL > 0 {
		asynqOpts = append(asynqOpts, asynq.Unique(opts.UniqueTTL))
	}

	info, err := c.asynqClient.EnqueueContext(ctx, task, asynqOpts...)
	if err != nil {
		log.Error("Failed to enqueue task",
			zap.String("task_type", taskType),
			zap.String("queue", queue),
			zap.Error(err))
		return nil, err
	}

	scheduledAt := time.Now()
	if opts.Delay > 0 {
		scheduledAt = scheduledAt.Add(opts.Delay)
	}

	result := &EnqueueResult{
		TaskID:      info.ID,
		Queue:       info.Queue,
		ScheduledAt: scheduledAt,
	}

	log.Info("Task enqueued successfully",
		zap.String("task_type", taskType),
		zap.String("task_id", info.ID),
		zap.String("queue", info.Queue),
		zap.Time("scheduled_at", scheduledAt),
		zap.Int("max_retry", info.MaxRetry),
	)

	return result, nil
}