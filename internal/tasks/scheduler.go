package tasks

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/logger"
)

type Scheduler struct {
	scheduler *asynq.Scheduler
	client    *Client
	config    *config.Config
}

func NewScheduler(cfg *config.Config, client *Client) (*Scheduler, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		LogLevel: asynq.InfoLevel,
	})

	return &Scheduler{
		scheduler: scheduler,
		client:    client,
		config:    cfg,
	}, nil
}

func (s *Scheduler) Start() error {
	log := logger.Global()

	log.Info("Registering scheduled tasks")

	entryID, err := s.scheduler.Register("@every 1m", asynq.NewTask(TaskTypeGenerateReport, []byte(`{
		"report_id": "scheduled_system_report",
		"params": {
			"type": "system_health",
			"timestamp": "` + time.Now().Format(time.RFC3339) + `"
		}
	}`)), asynq.Queue(QueueDefault))
	if err != nil {
		log.Error("Failed to register scheduled task", zap.Error(err))
		return err
	}

	log.Info("Scheduled task registered",
		zap.String("entry_id", entryID),
		zap.String("task_type", TaskTypeGenerateReport),
		zap.String("schedule", "@every 1m"))

	log.Info("Starting scheduler")
	return s.scheduler.Start()
}

func (s *Scheduler) Stop() {
	log := logger.Global()
	log.Info("Shutting down scheduler")
	s.scheduler.Shutdown()
}

func (s *Scheduler) ScheduleOneTime(ctx context.Context, taskType string, payload interface{}, at time.Time, opts *EnqueueOptions) (*EnqueueResult, error) {
	delay := time.Until(at)
	if delay < 0 {
		delay = 0
	}
	
	return s.client.EnqueueDelayed(ctx, taskType, payload, delay, opts)
}