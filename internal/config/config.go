package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App    AppConfig
	HTTP   HTTPConfig
	Worker WorkerConfig
	Redis  RedisConfig
	Asynq  AsynqConfig
	ELK    ELKConfig
	Tasks  TaskConfig
}

type AppConfig struct {
	Name    string
	Env     string
	Version string
}

type HTTPConfig struct {
	Addr string
}

type WorkerConfig struct {
	MetricsAddr string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AsynqConfig struct {
	ConcurrencyCritical int
	ConcurrencyDefault  int
	ConcurrencyLow      int
	ClientQueueDefault  string
}

type ELKConfig struct {
	ElasticsearchURL   string
	KibanaURL          string
	LogstashHost       string
	FilebeatDockerEnable bool
}

type TaskConfig struct {
	IdempotencyTTL   time.Duration
	DefaultMaxRetry  int
	DefaultTimeout   time.Duration
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// In production or Docker, .env might not exist - that's OK
	}

	cfg := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "atlasq"),
			Env:     getEnv("APP_ENV", "local"),
			Version: getEnv("APP_VERSION", "0.1.0"),
		},
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", ":8080"),
		},
		Worker: WorkerConfig{
			MetricsAddr: getEnv("WORKER_METRICS_ADDR", ":9090"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Asynq: AsynqConfig{
			ConcurrencyCritical: getEnvInt("ASYNQ_CONCURRENCY_CRITICAL", 20),
			ConcurrencyDefault:  getEnvInt("ASYNQ_CONCURRENCY_DEFAULT", 10),
			ConcurrencyLow:      getEnvInt("ASYNQ_CONCURRENCY_LOW", 5),
			ClientQueueDefault:  getEnv("ASYNQ_CLIENT_QUEUE_DEFAULT", "default"),
		},
		ELK: ELKConfig{
			ElasticsearchURL:     getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
			KibanaURL:           getEnv("KIBANA_URL", "http://localhost:5601"),
			LogstashHost:        getEnv("LOGSTASH_HOST", "localhost:5044"),
			FilebeatDockerEnable: getEnvBool("FILEBEAT_DOCKER_ENABLE", false),
		},
		Tasks: TaskConfig{
			IdempotencyTTL:  getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),
			DefaultMaxRetry: getEnvInt("DEFAULT_MAX_RETRY", 10),
			DefaultTimeout:  getEnvDuration("DEFAULT_TASK_TIMEOUT", 60*time.Second),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}