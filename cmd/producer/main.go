package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/logger"
	"github.com/yourorg/atlasq/internal/tasks"
	"github.com/yourorg/atlasq/internal/web"
	"github.com/yourorg/atlasq/pkg/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.App.Name+"-producer", cfg.App.Version, cfg.App.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Starting AtlasQ Producer",
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
		zap.String("addr", cfg.HTTP.Addr))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	log.Info("Connected to Redis", zap.String("addr", cfg.Redis.Addr))

	tasksClient, err := tasks.NewClient(cfg)
	if err != nil {
		log.Fatal("Failed to create tasks client", zap.Error(err))
	}
	defer tasksClient.Close()

	idempotencyMgr := tasks.NewIdempotencyManager(redisClient, cfg.Tasks.IdempotencyTTL)

	metrics := observability.NewMetrics()
	_ = metrics

	server := web.NewServer(cfg, tasksClient, idempotencyMgr)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			log.Error("HTTP server error", zap.Error(err))
			done <- syscall.SIGTERM
		}
	}()

	log.Info("Producer service started successfully")

	<-done
	log.Info("Shutting down producer service")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	if err := redisClient.Close(); err != nil {
		log.Error("Redis client close error", zap.Error(err))
	}

	log.Info("Producer service shutdown complete")
}