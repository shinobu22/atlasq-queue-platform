package middleware

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/logger"
)

func AsynqLogging() asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()

			// Generate a simple task ID since GetTaskInfo may not be available in this version
			taskID := t.Type() + "-" + start.Format("20060102150405")
			ctx = logger.WithTaskID(ctx, taskID)

			log := logger.FromContext(ctx)
			log.Info("Task started",
				zap.String("task_type", t.Type()),
				zap.String("task_id", taskID),
			)

			err := h.ProcessTask(ctx, t)
			duration := time.Since(start)

			if err != nil {
				log.Error("Task failed",
					zap.String("task_type", t.Type()),
					zap.String("task_id", taskID),
					zap.Int64("latency_ms", duration.Milliseconds()),
					zap.Error(err),
				)
			} else {
				log.Info("Task completed",
					zap.String("task_type", t.Type()),
					zap.String("task_id", taskID),
					zap.Int64("latency_ms", duration.Milliseconds()),
				)
			}

			return err
		})
	}
}