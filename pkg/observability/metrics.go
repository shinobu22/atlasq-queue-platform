package observability

import (
	"expvar"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	httpRequestsTotal     = expvar.NewInt("http_requests_total")
	httpRequestDuration   = expvar.NewInt("http_request_duration_ms")
	tasksEnqueuedTotal    = expvar.NewInt("tasks_enqueued_total")
	tasksProcessedTotal   = expvar.NewInt("tasks_processed_total")
	tasksFailedTotal      = expvar.NewInt("tasks_failed_total")
	workerGoroutines      = expvar.NewInt("worker_goroutines")
	redisConnections      = expvar.NewInt("redis_connections")
)

type Metrics struct {
	httpRequests     int64
	httpDuration     int64
	tasksEnqueued    int64
	tasksProcessed   int64
	tasksFailed      int64
}

func NewMetrics() *Metrics {
	m := &Metrics{}
	
	expvar.Publish("runtime", expvar.Func(func() interface{} {
		return map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]interface{}{
				"alloc":      getBytesValue("alloc"),
				"total_alloc": getBytesValue("total_alloc"),
				"sys":        getBytesValue("sys"),
				"num_gc":     getGCValue(),
			},
		}
	}))
	
	return m
}

func (m *Metrics) IncHTTPRequests() {
	atomic.AddInt64(&m.httpRequests, 1)
	httpRequestsTotal.Set(atomic.LoadInt64(&m.httpRequests))
}

func (m *Metrics) RecordHTTPDuration(duration time.Duration) {
	ms := duration.Milliseconds()
	atomic.AddInt64(&m.httpDuration, ms)
	httpRequestDuration.Set(atomic.LoadInt64(&m.httpDuration))
}

func (m *Metrics) IncTasksEnqueued() {
	atomic.AddInt64(&m.tasksEnqueued, 1)
	tasksEnqueuedTotal.Set(atomic.LoadInt64(&m.tasksEnqueued))
}

func (m *Metrics) IncTasksProcessed() {
	atomic.AddInt64(&m.tasksProcessed, 1)
	tasksProcessedTotal.Set(atomic.LoadInt64(&m.tasksProcessed))
}

func (m *Metrics) IncTasksFailed() {
	atomic.AddInt64(&m.tasksFailed, 1)
	tasksFailedTotal.Set(atomic.LoadInt64(&m.tasksFailed))
}

func (m *Metrics) SetWorkerGoroutines(count int64) {
	workerGoroutines.Set(count)
}

func (m *Metrics) SetRedisConnections(count int64) {
	redisConnections.Set(count)
}

func getBytesValue(key string) uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	switch key {
	case "alloc":
		return stats.Alloc
	case "total_alloc":
		return stats.TotalAlloc
	case "sys":
		return stats.Sys
	default:
		return 0
	}
}

func getGCValue() uint32 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.NumGC
}