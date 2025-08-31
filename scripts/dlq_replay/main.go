package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	var (
		redisAddr     = flag.String("redis-addr", getEnv("REDIS_ADDR", "localhost:6379"), "Redis address")
		redisPassword = flag.String("redis-password", getEnv("REDIS_PASSWORD", ""), "Redis password")
		redisDB       = flag.Int("redis-db", 0, "Redis database number")
		queueName     = flag.String("queue", "default", "Queue name to operate on")
		dryRun        = flag.Bool("dry-run", false, "Show what would be done without actually doing it")
	)
	flag.Parse()

	godotenv.Load()

	redisOpt := asynq.RedisClientOpt{
		Addr:     *redisAddr,
		Password: *redisPassword,
		DB:       *redisDB,
	}

	inspector := asynq.NewInspector(redisOpt)
	defer inspector.Close()

	fmt.Printf("Checking queue '%s' for archived tasks...\n", *queueName)

	if *dryRun {
		fmt.Println("--- DRY RUN MODE ---")
		fmt.Printf("Would attempt to replay all archived tasks in queue '%s'\n", *queueName)
		fmt.Println("Use without --dry-run to actually replay the tasks")
		return
	}

	// Try to run all archived tasks (move them back to pending)
	fmt.Printf("Attempting to replay archived tasks in queue '%s'...\n", *queueName)
	
	n, err := inspector.RunAllArchivedTasks(*queueName)
	if err != nil {
		log.Fatalf("Failed to replay archived tasks: %v", err)
	}

	if n == 0 {
		fmt.Println("No archived tasks found to replay")
	} else {
		fmt.Printf("Successfully moved %d archived tasks back to pending state\n", n)
	}

	fmt.Printf("\nReplay summary:\n")
	fmt.Printf("  Queue: %s\n", *queueName)
	fmt.Printf("  Tasks replayed: %d\n", n)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}