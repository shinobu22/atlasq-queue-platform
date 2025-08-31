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

type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

func (h *ReportHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	log := logger.FromContext(ctx)
	
	var payload tasks.GenerateReportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Error("Failed to unmarshal report payload", zap.Error(err))
		return tasks.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	if err := payload.Validate(); err != nil {
		log.Error("Report payload validation failed", zap.Error(err))
		return err
	}

	log.Info("Processing generate report task",
		zap.String("report_id", payload.ReportID),
		zap.Any("params", payload.Params))

	if err := h.generateReport(ctx, &payload); err != nil {
		log.Error("Failed to generate report", zap.Error(err))
		return err
	}

	log.Info("Report generated successfully",
		zap.String("report_id", payload.ReportID))

	return nil
}

func (h *ReportHandler) generateReport(ctx context.Context, payload *tasks.GenerateReportPayload) error {
	log := logger.FromContext(ctx)

	processingTime := time.Duration(rand.Intn(2000)+500) * time.Millisecond
	log.Debug("Simulating report generation",
		zap.String("report_id", payload.ReportID),
		zap.Duration("estimated_time", processingTime))

	if rand.Float32() < 0.02 {
		return tasks.NewPermanentError(fmt.Errorf("invalid report configuration for ID: %s", payload.ReportID))
	}

	if rand.Float32() < 0.08 {
		return tasks.NewRetryableError(fmt.Errorf("database temporarily unavailable"))
	}

	if payload.ReportID == "test-permanent-failure" {
		return tasks.NewPermanentError(fmt.Errorf("test permanent failure"))
	}

	if payload.ReportID == "test-retryable-failure" {
		return tasks.NewRetryableError(fmt.Errorf("test retryable failure"))
	}

	select {
	case <-ctx.Done():
		return tasks.NewRetryableError(fmt.Errorf("report generation cancelled: %w", ctx.Err()))
	case <-time.After(processingTime):
	}

	h.simulateReportSteps(log, payload)

	log.Debug("Simulated report generation completed",
		zap.String("report_id", payload.ReportID),
		zap.Duration("actual_time", processingTime))

	return nil
}

func (h *ReportHandler) simulateReportSteps(log *zap.Logger, payload *tasks.GenerateReportPayload) {
	steps := []string{
		"Collecting data sources",
		"Processing raw data",
		"Applying filters and transformations",
		"Generating visualizations",
		"Creating summary statistics",
		"Formatting final report",
	}

	for i, step := range steps {
		time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)
		log.Debug("Report generation step",
			zap.String("report_id", payload.ReportID),
			zap.Int("step", i+1),
			zap.String("description", step),
			zap.Int("total_steps", len(steps)))
	}
}