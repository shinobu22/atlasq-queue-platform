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

type EmailHandler struct{}

func NewEmailHandler() *EmailHandler {
	return &EmailHandler{}
}

func (h *EmailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	log := logger.FromContext(ctx)
	
	var payload tasks.SendEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Error("Failed to unmarshal email payload", zap.Error(err))
		return tasks.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	if err := payload.Validate(); err != nil {
		log.Error("Email payload validation failed", zap.Error(err))
		return err
	}

	log.Info("Processing send email task",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
		zap.Int("body_length", len(payload.Body)))

	if err := h.sendEmail(ctx, &payload); err != nil {
		log.Error("Failed to send email", zap.Error(err))
		return err
	}

	log.Info("Email sent successfully",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject))

	return nil
}

func (h *EmailHandler) sendEmail(ctx context.Context, payload *tasks.SendEmailPayload) error {
	log := logger.FromContext(ctx)

	simulateProcessingTime := time.Duration(rand.Intn(500)+100) * time.Millisecond
	time.Sleep(simulateProcessingTime)

	if rand.Float32() < 0.05 {
		return tasks.NewPermanentError(fmt.Errorf("invalid email address: %s", payload.To))
	}

	if rand.Float32() < 0.1 {
		return tasks.NewRetryableError(fmt.Errorf("SMTP server temporarily unavailable"))
	}

	if payload.Subject == "test-permanent-failure" {
		return tasks.NewPermanentError(fmt.Errorf("test permanent failure"))
	}

	if payload.Subject == "test-retryable-failure" {
		return tasks.NewRetryableError(fmt.Errorf("test retryable failure"))
	}

	log.Debug("Simulated email sending",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
		zap.Duration("processing_time", simulateProcessingTime))

	return nil
}