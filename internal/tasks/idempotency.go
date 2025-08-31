package tasks

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/logger"
)

type IdempotencyManager struct {
	redisClient *redis.Client
	defaultTTL  time.Duration
}

type IdempotencyResult struct {
	IsNew      bool   `json:"is_new"`
	ExistingID string `json:"existing_id,omitempty"`
}

func NewIdempotencyManager(redisClient *redis.Client, defaultTTL time.Duration) *IdempotencyManager {
	return &IdempotencyManager{
		redisClient: redisClient,
		defaultTTL:  defaultTTL,
	}
}

func (im *IdempotencyManager) CheckOrSet(ctx context.Context, key string, taskID string, ttl time.Duration) (*IdempotencyResult, error) {
	log := logger.FromContext(ctx)

	if ttl == 0 {
		ttl = im.defaultTTL
	}

	idempotencyKey := fmt.Sprintf("idem:%s", key)

	result, err := im.redisClient.SetNX(ctx, idempotencyKey, taskID, ttl).Result()
	if err != nil {
		log.Error("Failed to check idempotency key",
			zap.String("key", idempotencyKey),
			zap.Error(err))
		return nil, err
	}

	if result {
		log.Debug("New idempotency key set",
			zap.String("key", idempotencyKey),
			zap.String("task_id", taskID),
			zap.Duration("ttl", ttl))
		
		return &IdempotencyResult{
			IsNew: true,
		}, nil
	}

	existingID, err := im.redisClient.Get(ctx, idempotencyKey).Result()
	if err != nil {
		if err == redis.Nil {
			log.Warn("Idempotency key expired between SetNX and Get",
				zap.String("key", idempotencyKey))
			
			result, err := im.redisClient.SetNX(ctx, idempotencyKey, taskID, ttl).Result()
			if err != nil {
				return nil, err
			}
			
			return &IdempotencyResult{
				IsNew: result,
				ExistingID: func() string {
					if result {
						return ""
					}
					return "unknown"
				}(),
			}, nil
		}
		
		log.Error("Failed to get existing idempotency key",
			zap.String("key", idempotencyKey),
			zap.Error(err))
		return nil, err
	}

	log.Debug("Idempotency key already exists",
		zap.String("key", idempotencyKey),
		zap.String("existing_id", existingID),
		zap.String("new_id", taskID))

	return &IdempotencyResult{
		IsNew:      false,
		ExistingID: existingID,
	}, nil
}

func (im *IdempotencyManager) GenerateKey(taskType string, payload interface{}) string {
	data := fmt.Sprintf("%s:%+v", taskType, payload)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func (im *IdempotencyManager) Delete(ctx context.Context, key string) error {
	log := logger.FromContext(ctx)
	
	idempotencyKey := fmt.Sprintf("idem:%s", key)
	
	result, err := im.redisClient.Del(ctx, idempotencyKey).Result()
	if err != nil {
		log.Error("Failed to delete idempotency key",
			zap.String("key", idempotencyKey),
			zap.Error(err))
		return err
	}

	log.Debug("Idempotency key deleted",
		zap.String("key", idempotencyKey),
		zap.Int64("deleted_count", result))

	return nil
}