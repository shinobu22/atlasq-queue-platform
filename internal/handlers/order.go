package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/logger"
	"github.com/yourorg/atlasq/internal/tasks"
)

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

func (h *OrderHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	log := logger.FromContext(ctx)
	
	var payload tasks.ProcessOrderPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Error("Failed to unmarshal order payload", zap.Error(err))
		return tasks.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	if err := payload.Validate(); err != nil {
		log.Error("Order payload validation failed", zap.Error(err))
		return err
	}

	log.Info("Processing order task",
		zap.String("order_number", payload.OrderNumber),
		zap.Int("quantity", payload.Quantity))

	if err := h.processOrder(ctx, &payload); err != nil {
		log.Error("Failed to process order", zap.Error(err))
		return err
	}

	log.Info("Order processed successfully",
		zap.String("order_number", payload.OrderNumber),
		zap.Int("quantity", payload.Quantity))

	return nil
}

func (h *OrderHandler) processOrder(ctx context.Context, payload *tasks.ProcessOrderPayload) error {
	log := logger.FromContext(ctx)

	// Simulate order processing time
	simulateProcessingTime := time.Duration(rand.Intn(1000)+200) * time.Millisecond
	time.Sleep(simulateProcessingTime)

	// Simulate occasional failures for demonstration
	if rand.Float32() < 0.02 {
		return tasks.NewPermanentError(fmt.Errorf("invalid order number: %s", payload.OrderNumber))
	}

	if rand.Float32() < 0.05 {
		return tasks.NewRetryableError(fmt.Errorf("inventory service temporarily unavailable"))
	}

	// Special test cases
	if payload.OrderNumber == "test-permanent-failure" {
		return tasks.NewPermanentError(fmt.Errorf("test permanent failure"))
	}

	if payload.OrderNumber == "test-retryable-failure" {
		return tasks.NewRetryableError(fmt.Errorf("test retryable failure"))
	}

	log.Debug("Simulated order processing",
		zap.String("order_number", payload.OrderNumber),
		zap.Int("quantity", payload.Quantity),
		zap.Duration("processing_time", simulateProcessingTime))

	return nil
}