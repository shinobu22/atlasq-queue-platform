# AtlasQ

A production-ready queue platform built with **Go + Asynq + Redis**, featuring comprehensive observability through the **ELK stack** (Elasticsearch, Logstash, Kibana) and **Asynqmon** for queue monitoring.

## Features

- 🚀 **High-performance task processing** with Redis-backed Asynq
- 📊 **Multi-queue support** (critical, default, low) with weighted processing  
- 🔄 **Intelligent retry logic** with exponential backoff and jitter
- 🎯 **Task idempotency** with Redis-based deduplication
- ⏰ **Flexible scheduling** (immediate, delayed, cron-based)
- 🔍 **Full observability** with structured JSON logs shipped to ELK
- 📈 **Real-time monitoring** with Asynqmon dashboard
- 🐳 **Container-ready** with Docker Compose orchestration
- ☸️  **Kubernetes manifests** for scalable deployment
- 🧪 **Load testing** with k6 scenarios

## Quick Start

1. **Clone and start services:**
   ```bash
   git clone <repository-url>
   cd atlasq
   make demo
   ```

2. **Enqueue a task:**
   ```bash
   curl -X POST http://localhost:8080/enqueue/send_email \
     -H "Content-Type: application/json" \
     -d '{
       "payload": {
         "to": "user@example.com",
         "subject": "Welcome to AtlasQ!",
         "body": "Your queue platform is ready."
       }
     }'
   ```

3. **Monitor queues:**
   - **Asynqmon:** http://localhost:8081 (queue dashboard)
   - **Kibana:** http://localhost:5601 (log analysis)
   - **Worker metrics:** http://localhost:9090/metrics

## Architecture

```
┌─────────────┐    HTTP     ┌─────────────┐    Redis    ┌─────────────┐
│   Clients   │────────────▶│  Producer   │────────────▶│   Redis     │
└─────────────┘             │   (HTTP)    │             │  (Queue)    │
                            └─────────────┘             └─────────────┘
                                                               │
                            ┌─────────────┐    Redis    ┌─────┴─────┐
                            │  Worker(s)  │◀────────────│ Asynqmon  │
                            │ (Consumer)  │             │(Monitor)  │
                            └─────────────┘             └───────────┘
                                   │
                            ┌─────────────┐
                            │     ELK     │
                            │   (Logs)    │
                            └─────────────┘
```

## Task Types

### Send Email (`send_email`)
```json
{
  "payload": {
    "to": "user@example.com",
    "subject": "Email subject",
    "body": "Email content"
  }
}
```

### Generate Report (`generate_report`)
```json
{
  "payload": {
    "report_id": "monthly-sales-2024",
    "params": {
      "type": "sales",
      "period": "monthly",
      "format": "pdf"
    }
  }
}
```

### Process Order (`process_order`)
```json
{
  "payload": {
    "order_number": "ORD-2024-001",
    "quantity": 5
  }
}
```

## API Reference

### Enqueue Tasks
**POST** `/enqueue/{task_type}`

**Query Parameters:**
- `queue`: Target queue (`critical`, `default`, `low`)
- `delay`: Delay execution (`10s`, `5m`, `2h`)  
- `max_retry`: Override retry limit (0-50)
- `unique_ttl`: Prevent duplicates for duration (`1h`, `24h`)

**Headers:**
- `X-Idempotency-Key`: Prevent duplicate submissions

**Example:**
```bash
curl -X POST 'http://localhost:8080/enqueue/send_email?queue=critical&delay=30s' \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: user-welcome-123" \
  -d '{"payload": {"to": "user@example.com", "subject": "Welcome!", "body": "Hello!"}}'
```

### Orders API
**POST** `/v1/orders`

Fast order processing endpoint that immediately enqueues orders for async processing.

**Request Body:**
```json
{
  "order_number": "ORD-2024-001",
  "quantity": 5
}
```

**Response:**
```json
{
  "status": "success"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "order_number": "ORD-2024-001",
    "quantity": 10
  }'
```

**Load Testing:**
```bash
# Run k6 load test for orders endpoint
k6 run load_test.js
```

### Health Endpoints
- **GET** `/healthz` - Basic health check
- **GET** `/readyz` - Readiness probe

## Configuration

Environment variables (see `.env.example`):

```bash
# Application
APP_NAME=atlasq
APP_ENV=local
APP_VERSION=0.1.0

# Services  
HTTP_ADDR=:8080
WORKER_METRICS_ADDR=:9090

# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=atlasqredis
REDIS_DB=0

# Queue Concurrency
ASYNQ_CONCURRENCY_CRITICAL=20
ASYNQ_CONCURRENCY_DEFAULT=10
ASYNQ_CONCURRENCY_LOW=5

# Task Settings
DEFAULT_MAX_RETRY=10
DEFAULT_TASK_TIMEOUT=60s
IDEMPOTENCY_TTL=24h
```

## Development

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Make (optional, for convenience)

### Commands
```bash
# Setup
make deps              # Download dependencies
make env               # Create .env from example

# Development
make dev-redis         # Start Redis only
make dev-producer      # Run producer locally
make dev-worker        # Run worker locally

# Testing
make test              # Run tests
make lint              # Run linters  
make fmt               # Format code

# Docker
make up                # Start all services
make down              # Stop all services
make logs              # Follow main logs
make status            # Service status

# Tasks
make seed              # Enqueue demo tasks
make dlq               # Replay dead tasks
make dlq-dry           # Show DLQ contents

# Load Testing
make k6                # Full load test
make k6-smoke          # Light smoke test

# Health
make health            # Check all services
make metrics           # Show worker metrics
```

## Queue Configuration

**Queue Priorities & Concurrency:**
- **Critical:** Weight 5, Concurrency 20 (urgent tasks)
- **Default:** Weight 3, Concurrency 10 (normal tasks)  
- **Low:** Weight 1, Concurrency 5 (background tasks)

**Retry Policy:**
- Exponential backoff with jitter
- Configurable max retries (default: 10)
- Permanent vs retryable error classification
- Dead Letter Queue (DLQ) for failed tasks

## Observability

### Structured Logging
JSON logs with contextual fields:
```json
{
  "ts": "2024-01-15T10:30:00Z",
  "level": "info", 
  "service": "atlasq-worker",
  "version": "0.1.0",
  "env": "production",
  "request_id": "req-123",
  "task_id": "task-456", 
  "task_type": "send_email",
  "queue": "default",
  "attempt": 1,
  "latency_ms": 250,
  "msg": "Task completed"
}
```

### ELK Stack Integration
- **Filebeat:** Collects container logs
- **Logstash:** Parses and enriches logs  
- **Elasticsearch:** Indexes with ILM policies
- **Kibana:** Visualizes logs and metrics

### Metrics (Worker)
Exposed at `:9090/metrics` (expvar format):
- HTTP request counts/durations
- Task processing stats
- Runtime metrics (goroutines, memory)

## Production Deployment

### Docker Production
```bash
# Build images
make docker-build

# Deploy with production overrides
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Kubernetes
```bash
# Apply manifests (development example)
kubectl apply -f deployments/k8s/

# Scale workers
kubectl scale deployment worker --replicas=5
```

### Security Hardening

**⚠️ Production Checklist:**

1. **Redis Security:**
   - Enable AUTH with strong password
   - Use TLS encryption
   - Network policies/firewall rules
   - Regular security updates

2. **Application Security:**
   - Run as non-root user
   - Resource limits & quotas
   - Input validation & sanitization
   - Rate limiting
   - Secrets management (not env vars)

3. **Network Security:**
   - Private networks/VPCs
   - Ingress with TLS termination
   - Service mesh (optional)
   - Network policies

4. **Monitoring & Alerting:**
   - Prometheus metrics
   - Grafana dashboards  
   - Error rate alerts
   - Resource usage alerts

5. **ELK Security:**
   - Enable X-Pack security
   - RBAC and authentication
   - TLS between components
   - Index lifecycle management

## Troubleshooting

### Common Issues

**Services won't start:**
```bash
# Check logs
make logs

# Verify Redis connectivity  
docker-compose exec producer redis-cli -h redis -a atlasqredis ping

# Check service health
make health
```

**Tasks not processing:**
```bash
# Check worker logs
docker-compose logs worker

# Inspect queues
# Visit http://localhost:8081

# Check Redis directly
docker-compose exec redis redis-cli -a atlasqredis
> LLEN asynq:queue:default
```

**Missing logs in Kibana:**
```bash
# Check Filebeat status
docker-compose logs filebeat

# Verify Logstash pipeline
docker-compose logs logstash

# Check Elasticsearch indices
curl http://localhost:9200/_cat/indices?v
```

**High memory usage:**
```bash
# Check worker metrics
curl http://localhost:9090/metrics

# Scale workers horizontally
docker-compose up -d --scale worker=3
```

### Log Analysis Queries

**Kibana KQL Examples:**
```kql
# Failed tasks only
level: "error" AND task_type: *

# High latency requests  
latency_ms: >1000

# Specific queue activity
queue: "critical" AND level: "info"

# Error patterns
msg: *timeout* OR msg: *connection*
```

## Dead Letter Queue Management

**Replay failed tasks:**
```bash
# Show DLQ contents
make dlq-dry

# Replay all to default queue
make dlq

# Replay with custom options
go run scripts/dlq_replay.go -target-queue=low -max-tasks=50
```

**DLQ Monitoring:**
- Failed tasks automatically move to DLQ after max retries
- Monitor DLQ size in Asynqmon
- Set up alerts for DLQ growth

## Load Testing

The included k6 script tests realistic workloads:

```bash
# Full load test (ramps up to 20 VUs)
make k6

# Custom parameters
BASE_URL=https://your-domain.com make k6

# Results analysis
cat k6-results.json | jq '.metrics.http_req_duration.values'
```

## License

[Your License Here]

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality  
4. Ensure CI passes
5. Submit a pull request

---

**AtlasQ** - Production-ready task queuing made simple 🚀