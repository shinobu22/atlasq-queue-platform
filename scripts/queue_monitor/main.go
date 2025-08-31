package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type QueueStats struct {
	QueueName     string    `json:"queue_name"`
	PendingTasks  int64     `json:"pending_tasks"`
	ActiveTasks   int64     `json:"active_tasks"`
	ScheduledTasks int64    `json:"scheduled_tasks"`
	RetryTasks    int64     `json:"retry_tasks"`
	ArchivedTasks int64     `json:"archived_tasks"`
	CompletedTasks int64    `json:"completed_tasks"`
	FailedTasks   int64     `json:"failed_tasks"`
	Timestamp     time.Time `json:"timestamp"`
}

type OverallStats struct {
	Queues       []QueueStats `json:"queues"`
	TotalPending int64        `json:"total_pending"`
	TotalActive  int64        `json:"total_active"`
	Timestamp    time.Time    `json:"timestamp"`
}

func main() {
	var (
		redisAddr     = flag.String("redis-addr", getEnv("REDIS_ADDR", "localhost:6379"), "Redis address")
		redisPassword = flag.String("redis-password", getEnv("REDIS_PASSWORD", ""), "Redis password")
		redisDB       = flag.Int("redis-db", 0, "Redis database number")
		port          = flag.String("port", "8082", "HTTP port for web interface")
		interval      = flag.Duration("interval", 5*time.Second, "Update interval")
	)
	flag.Parse()

	godotenv.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     *redisAddr,
		Password: *redisPassword,
		DB:       *redisDB,
	})

	// Test connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	monitor := &QueueMonitor{
		client:   redisClient,
		interval: *interval,
	}

	// Set up HTTP server
	http.HandleFunc("/", monitor.handleHome)
	http.HandleFunc("/api/stats", monitor.handleStats)
	http.HandleFunc("/api/queues", monitor.handleQueues)

	fmt.Printf("AtlasQ Queue Monitor starting on http://localhost:%s\n", *port)
	fmt.Printf("Redis: %s, Update interval: %v\n", *redisAddr, *interval)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

type QueueMonitor struct {
	client   *redis.Client
	interval time.Duration
}

func (m *QueueMonitor) handleHome(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>AtlasQ Queue Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; text-align: center; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin: 20px 0; }
        .queue-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .queue-name { font-size: 18px; font-weight: bold; margin-bottom: 10px; color: #2c3e50; }
        .stat-row { display: flex; justify-content: space-between; margin: 8px 0; padding: 4px 0; }
        .stat-label { color: #666; }
        .stat-value { font-weight: bold; }
        .pending { color: #f39c12; }
        .active { color: #27ae60; }
        .failed { color: #e74c3c; }
        .completed { color: #3498db; }
        .refresh { text-align: center; margin: 20px 0; }
        .btn { background: #3498db; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        .btn:hover { background: #2980b9; }
        .timestamp { text-align: center; color: #666; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 AtlasQ Queue Monitor</h1>
        <div class="refresh">
            <button class="btn" onclick="location.reload()">Refresh Stats</button>
        </div>
        <div id="stats-container">Loading...</div>
        <div class="timestamp" id="timestamp"></div>
    </div>
    
    <script>
        function loadStats() {
            fetch('/api/stats')
                .then(response => response.json())
                .then(data => {
                    const container = document.getElementById('stats-container');
                    const timestamp = document.getElementById('timestamp');
                    
                    let html = '<div class="stats-grid">';
                    
                    data.queues.forEach(queue => {
                        html += '<div class="queue-card">';
                        html += '<div class="queue-name">' + queue.queue_name + '</div>';
                        html += '<div class="stat-row"><span class="stat-label">Pending:</span><span class="stat-value pending">' + queue.pending_tasks + '</span></div>';
                        html += '<div class="stat-row"><span class="stat-label">Active:</span><span class="stat-value active">' + queue.active_tasks + '</span></div>';
                        html += '<div class="stat-row"><span class="stat-label">Scheduled:</span><span class="stat-value">' + queue.scheduled_tasks + '</span></div>';
                        html += '<div class="stat-row"><span class="stat-label">Retry:</span><span class="stat-value">' + queue.retry_tasks + '</span></div>';
                        html += '<div class="stat-row"><span class="stat-label">Archived:</span><span class="stat-value failed">' + queue.archived_tasks + '</span></div>';
                        html += '</div>';
                    });
                    
                    html += '</div>';
                    container.innerHTML = html;
                    timestamp.innerHTML = 'Last updated: ' + new Date(data.timestamp).toLocaleString();
                })
                .catch(err => {
                    document.getElementById('stats-container').innerHTML = '<p style="color: red;">Error loading stats: ' + err + '</p>';
                });
        }
        
        // Load stats immediately
        loadStats();
        
        // Auto-refresh every 5 seconds
        setInterval(loadStats, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (m *QueueMonitor) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	queues := []string{"critical", "default", "low"}
	
	stats := OverallStats{
		Queues:    make([]QueueStats, 0),
		Timestamp: time.Now(),
	}

	for _, queueName := range queues {
		queueStats := QueueStats{
			QueueName: queueName,
			Timestamp: time.Now(),
		}

		// Get pending tasks
		pendingKey := fmt.Sprintf("asynq:queue:%s", queueName)
		pending, _ := m.client.LLen(ctx, pendingKey).Result()
		queueStats.PendingTasks = pending
		stats.TotalPending += pending

		// Get active tasks
		activeKey := fmt.Sprintf("asynq:active:%s", queueName)
		active, _ := m.client.ZCard(ctx, activeKey).Result()
		queueStats.ActiveTasks = active
		stats.TotalActive += active

		// Get scheduled tasks
		scheduledKey := fmt.Sprintf("asynq:scheduled:%s", queueName)
		scheduled, _ := m.client.ZCard(ctx, scheduledKey).Result()
		queueStats.ScheduledTasks = scheduled

		// Get retry tasks
		retryKey := fmt.Sprintf("asynq:retry:%s", queueName)
		retry, _ := m.client.ZCard(ctx, retryKey).Result()
		queueStats.RetryTasks = retry

		// Get archived tasks
		archivedKey := fmt.Sprintf("asynq:archived:%s", queueName)
		archived, _ := m.client.ZCard(ctx, archivedKey).Result()
		queueStats.ArchivedTasks = archived

		stats.Queues = append(stats.Queues, queueStats)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (m *QueueMonitor) handleQueues(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	
	// Get all keys related to asynq
	keys, err := m.client.Keys(ctx, "asynq:*").Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make(map[string]interface{})
	for _, key := range keys {
		keyType, _ := m.client.Type(ctx, key).Result()
		
		switch keyType {
		case "list":
			length, _ := m.client.LLen(ctx, key).Result()
			result[key] = map[string]interface{}{"type": "list", "length": length}
		case "zset":
			card, _ := m.client.ZCard(ctx, key).Result()
			result[key] = map[string]interface{}{"type": "zset", "cardinality": card}
		case "string":
			val, _ := m.client.Get(ctx, key).Result()
			result[key] = map[string]interface{}{"type": "string", "value": val}
		case "hash":
			hlen, _ := m.client.HLen(ctx, key).Result()
			result[key] = map[string]interface{}{"type": "hash", "length": hlen}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}