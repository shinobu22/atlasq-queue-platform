.PHONY: help build test lint fmt clean up down logs seed dlq k6 docker-build deps

# Variables
DOCKER_COMPOSE := docker compose -f deployments/docker/docker-compose.yml
PRODUCER_BINARY := bin/producer
WORKER_BINARY := bin/worker
SCRIPTS_DIR := scripts

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
deps: ## Download and verify Go module dependencies
	go mod download
	go mod verify

build: ## Build all binaries
	@mkdir -p bin
	go build -o $(PRODUCER_BINARY) ./cmd/producer
	go build -o $(WORKER_BINARY) ./cmd/worker
	@echo "Built binaries in bin/"

test: ## Run tests
	go test -v -race -cover ./...

lint: ## Run linters
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, using go vet only"; \
	fi

fmt: ## Format code
	go fmt ./...

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

# Docker Compose Operations
up: ## Start all services with docker-compose
	$(DOCKER_COMPOSE) up -d
	@echo "Services started. Access points:"
	@echo "  Producer API: http://localhost:8080"
	@echo "  Worker Metrics: http://localhost:9090/metrics"
	@echo "  Asynqmon: http://localhost:8081"
	@echo "  Kibana: http://localhost:5601"
	@echo "  Elasticsearch: http://localhost:9200"

down: ## Stop all services and remove volumes
	$(DOCKER_COMPOSE) down -v

logs: ## Follow logs for main services
	$(DOCKER_COMPOSE) logs -f producer worker asynqmon

logs-all: ## Follow logs for all services
	$(DOCKER_COMPOSE) logs -f

status: ## Show service status
	$(DOCKER_COMPOSE) ps

restart: ## Restart all services
	$(DOCKER_COMPOSE) restart

# Utility Scripts
seed: ## Enqueue demo tasks
	@if [ ! -f .env ]; then cp .env.example .env; fi
	go run $(SCRIPTS_DIR)/seed/main.go -email-count=20 -report-count=10 -with-failures -with-delays -verbose

seed-simple: ## Enqueue simple demo tasks
	@if [ ! -f .env ]; then cp .env.example .env; fi
	go run $(SCRIPTS_DIR)/seed/main.go -email-count=5 -report-count=3

dlq: ## Replay dead letter queue tasks
	go run $(SCRIPTS_DIR)/dlq_replay/main.go -queue=default

dlq-dry: ## Show DLQ tasks without replaying
	go run $(SCRIPTS_DIR)/dlq_replay/main.go -dry-run

# Load Testing
k6: ## Run k6 load test
	docker run --rm -i --network host \
		-e BASE_URL=http://localhost:8080 \
		grafana/k6 run - < k6/enqueue_load_test.js

k6-smoke: ## Run k6 smoke test (light load)
	docker run --rm -i --network host \
		-e BASE_URL=http://localhost:8080 \
		grafana/k6 run --vus 2 --duration 30s - < k6/enqueue_load_test.js

# Docker Images
docker-build: ## Build Docker images
	docker build -f deployments/docker/Dockerfile.producer -t atlasq/producer:latest .
	docker build -f deployments/docker/Dockerfile.worker -t atlasq/worker:latest .

docker-push: docker-build ## Build and push Docker images
	docker push atlasq/producer:latest
	docker push atlasq/worker:latest

# Development Shortcuts
dev-producer: ## Run producer locally
	@if [ ! -f .env ]; then cp .env.example .env; fi
	go run ./cmd/producer

dev-worker: ## Run worker locally
	@if [ ! -f .env ]; then cp .env.example .env; fi
	go run ./cmd/worker

dev-redis: ## Start only Redis for local development
	$(DOCKER_COMPOSE) up -d redis

# Health Checks
health: ## Check service health
	@echo "Checking service health..."
	@curl -s http://localhost:8080/healthz | jq . || echo "Producer not responding"
	@curl -s http://localhost:9090/healthz | jq . || echo "Worker not responding"
	@curl -s http://localhost:9200/_cluster/health | jq . || echo "Elasticsearch not responding"

metrics: ## Show worker metrics
	@curl -s http://localhost:9090/metrics || echo "Metrics not available"

# Development Environment
env: ## Create .env file from example
	cp .env.example .env
	@echo "Created .env file. Edit it to customize configuration."

# Convenience targets for common workflows
demo: up seed ## Start services and seed with demo data
	@echo ""
	@echo "Demo environment ready!"
	@echo "Visit http://localhost:8081 to see queued tasks"
	@echo "Visit http://localhost:5601 to see logs in Kibana (may take a few minutes to start)"

quick-test: build test ## Quick build and test
	@echo "Quick test completed successfully"

full-test: deps test lint ## Full test suite including linting
	@echo "Full test suite completed successfully"