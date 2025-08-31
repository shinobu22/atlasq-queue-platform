package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	TaskIDKey    contextKey = "task_id"
)

var globalLogger *zap.Logger

func New(service, version, env string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	
	if env == "local" || env == "development" {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config.Encoding = "json"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	logger = logger.With(
		zap.String("service", service),
		zap.String("version", version),
		zap.String("env", env),
	)

	globalLogger = logger
	return logger, nil
}

func FromContext(ctx context.Context) *zap.Logger {
	if globalLogger == nil {
		panic("logger not initialized")
	}

	logger := globalLogger

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			logger = logger.With(zap.String("request_id", id))
		}
	}

	if taskID := ctx.Value(TaskIDKey); taskID != nil {
		if id, ok := taskID.(string); ok && id != "" {
			logger = logger.With(zap.String("task_id", id))
		}
	}

	return logger
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, TaskIDKey, taskID)
}

func Global() *zap.Logger {
	if globalLogger == nil {
		panic("logger not initialized")
	}
	return globalLogger
}