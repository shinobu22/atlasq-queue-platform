package main

import (
	"context"
	"errors"
	"expvar"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/handlers"
	"github.com/yourorg/atlasq/internal/logger"
	"github.com/yourorg/atlasq/internal/middleware"
	"github.com/yourorg/atlasq/internal/tasks"
	"github.com/yourorg/atlasq/pkg/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.App.Name+"-worker", cfg.App.Version, cfg.App.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Starting AtlasQ Worker",
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
		zap.String("metrics_addr", cfg.Worker.MetricsAddr))

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	queueConfig := map[string]int{
		tasks.QueueCritical: cfg.Asynq.ConcurrencyCritical,
		tasks.QueueDefault:  cfg.Asynq.ConcurrencyDefault,
		tasks.QueueLow:      cfg.Asynq.ConcurrencyLow,
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Asynq.ConcurrencyCritical + cfg.Asynq.ConcurrencyDefault + cfg.Asynq.ConcurrencyLow,
			Queues:      queueConfig,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log := logger.FromContext(ctx)
				
				var permanentErr tasks.PermanentError
				var retryableErr tasks.RetryableError
				
				if errors.As(err, &permanentErr) {
					log.Error("Task permanent failure - will not retry",
						zap.String("task_type", task.Type()),
						zap.Error(err))
				} else if errors.As(err, &retryableErr) {
					log.Warn("Task retryable failure",
						zap.String("task_type", task.Type()),
						zap.Error(err))
				} else {
					log.Error("Task unexpected error",
						zap.String("task_type", task.Type()),
						zap.Error(err))
				}
			}),
			IsFailure: func(err error) bool {
				var permanentErr tasks.PermanentError
				return errors.As(err, &permanentErr)
			},
			RetryDelayFunc: asynq.DefaultRetryDelayFunc,
		},
	)

	mux := asynq.NewServeMux()
	mux.Use(middleware.AsynqLogging())

	emailHandler := handlers.NewEmailHandler()
	reportHandler := handlers.NewReportHandler()
	orderHandler := handlers.NewOrderHandler()

	mux.HandleFunc(tasks.TaskTypeSendEmail, emailHandler.ProcessTask)
	mux.HandleFunc(tasks.TaskTypeGenerateReport, reportHandler.ProcessTask)
	mux.HandleFunc(tasks.TaskTypeProcessOrder, orderHandler.ProcessTask)

	metrics := observability.NewMetrics()
	_ = metrics

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", expvar.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"` + cfg.App.Name + `-worker","version":"` + cfg.App.Version + `"}`))
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"` + cfg.App.Name + `-worker","version":"` + cfg.App.Version + `"}`))
		})

		server := &http.Server{
			Addr:    cfg.Worker.MetricsAddr,
			Handler: mux,
		}

		log.Info("Starting metrics server", zap.String("addr", cfg.Worker.MetricsAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Metrics server error", zap.Error(err))
		}
	}()

	tasksClient, err := tasks.NewClient(cfg)
	if err != nil {
		log.Fatal("Failed to create tasks client", zap.Error(err))
	}
	defer tasksClient.Close()

	scheduler, err := tasks.NewScheduler(cfg, tasksClient)
	if err != nil {
		log.Fatal("Failed to create scheduler", zap.Error(err))
	}

	go func() {
		if err := scheduler.Start(); err != nil {
			log.Error("Scheduler error", zap.Error(err))
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting task processing",
		zap.Int("critical_concurrency", cfg.Asynq.ConcurrencyCritical),
		zap.Int("default_concurrency", cfg.Asynq.ConcurrencyDefault),
		zap.Int("low_concurrency", cfg.Asynq.ConcurrencyLow))

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Error("Worker server error", zap.Error(err))
			done <- syscall.SIGTERM
		}
	}()

	log.Info("Worker service started successfully")

	<-done
	log.Info("Shutting down worker service")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	scheduler.Stop()

	srv.Shutdown()

	select {
	case <-shutdownCtx.Done():
		log.Warn("Worker shutdown timed out")
	default:
		log.Info("Worker service shutdown complete")
	}
}