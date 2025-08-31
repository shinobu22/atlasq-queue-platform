package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/logger"
	"github.com/yourorg/atlasq/internal/tasks"
)

type EnqueueRequest struct {
	Payload interface{} `json:"payload"`
}

type EnqueueResponse struct {
	Enqueued    bool                    `json:"enqueued"`
	TaskID      string                  `json:"task_id"`
	Queue       string                  `json:"queue"`
	ScheduledAt time.Time               `json:"scheduled_at"`
	Existing    bool                    `json:"existing,omitempty"`
	Result      *tasks.EnqueueResult    `json:"result,omitempty"`
}

type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type OrderRequest struct {
	OrderNumber string `json:"order_number"`
	Quantity    int    `json:"quantity"`
}

type OrderResponse struct {
	Status string `json:"status"`
}

func setupRoutes(r chi.Router, cfg *config.Config, client *tasks.Client, idempotencyMgr *tasks.IdempotencyManager) {
	r.Route("/enqueue", func(r chi.Router) {
		r.Post("/{task}", handleEnqueue(cfg, client, idempotencyMgr))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/orders", handleOrders(cfg, client))
		r.Get("/stocks", handleStocks(cfg))
	})

	r.Get("/healthz", handleHealth(cfg, false))
	r.Get("/readyz", handleHealth(cfg, true))
}

func handleEnqueue(cfg *config.Config, client *tasks.Client, idempotencyMgr *tasks.IdempotencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		taskType := chi.URLParam(r, "task")
		if taskType == "" {
			writeErrorResponse(w, http.StatusBadRequest, "task type is required")
			return
		}

		if taskType != tasks.TaskTypeSendEmail && taskType != tasks.TaskTypeGenerateReport && taskType != tasks.TaskTypeProcessOrder {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("unknown task type: %s", taskType))
			return
		}

		var req EnqueueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode request body", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}

		opts, err := parseEnqueueOptions(r)
		if err != nil {
			log.Error("Failed to parse enqueue options", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		idempotencyKey := r.Header.Get("X-Idempotency-Key")
		var existing bool
		var taskID string

		if idempotencyKey != "" {
			idempotencyResult, err := idempotencyMgr.CheckOrSet(ctx, idempotencyKey, "", cfg.Tasks.IdempotencyTTL)
			if err != nil {
				log.Error("Failed to check idempotency", zap.Error(err))
				writeErrorResponse(w, http.StatusInternalServerError, "idempotency check failed")
				return
			}

			if !idempotencyResult.IsNew {
				existing = true
				taskID = idempotencyResult.ExistingID
				
				response := &EnqueueResponse{
					Enqueued: false,
					TaskID:   taskID,
					Existing: true,
				}
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		result, err := client.EnqueueNow(ctx, taskType, req.Payload, opts)
		if err != nil {
			log.Error("Failed to enqueue task", 
				zap.String("task_type", taskType),
				zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to enqueue task: %v", err))
			return
		}

		if idempotencyKey != "" && !existing {
			_, err := idempotencyMgr.CheckOrSet(ctx, idempotencyKey, result.TaskID, cfg.Tasks.IdempotencyTTL)
			if err != nil {
				log.Warn("Failed to update idempotency key with actual task ID", 
					zap.String("task_id", result.TaskID),
					zap.Error(err))
			}
		}

		response := &EnqueueResponse{
			Enqueued:    true,
			TaskID:      result.TaskID,
			Queue:       result.Queue,
			ScheduledAt: result.ScheduledAt,
			Result:      result,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func handleOrders(cfg *config.Config, client *tasks.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode order request body", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}

		// Create the payload for the task
		orderPayload := tasks.ProcessOrderPayload{
			OrderNumber: req.OrderNumber,
			Quantity:    req.Quantity,
		}

		// Enqueue the task immediately
		opts := &tasks.EnqueueOptions{
			Queue: tasks.QueueDefault, // Use default queue for orders
		}

		_, err := client.EnqueueNow(ctx, tasks.TaskTypeProcessOrder, orderPayload, opts)
		if err != nil {
			log.Error("Failed to enqueue order task", 
				zap.String("order_number", req.OrderNumber),
				zap.Int("quantity", req.Quantity),
				zap.Error(err))
			writeErrorResponse(w, http.StatusInternalServerError, "failed to process order")
			return
		}

		log.Info("Order enqueued successfully",
			zap.String("order_number", req.OrderNumber),
			zap.Int("quantity", req.Quantity))

		response := &OrderResponse{
			Status: "success",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func handleHealth(cfg *config.Config, readiness bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)
		
		if readiness {
			checks["redis"] = "ok"
		}

		response := &HealthResponse{
			Status:  "ok",
			Service: cfg.App.Name,
			Version: cfg.App.Version,
			Checks:  checks,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func parseEnqueueOptions(r *http.Request) (*tasks.EnqueueOptions, error) {
	opts := &tasks.EnqueueOptions{}

	if delayStr := r.URL.Query().Get("delay"); delayStr != "" {
		delay, err := time.ParseDuration(delayStr)
		if err != nil {
			return nil, fmt.Errorf("invalid delay format: %v", err)
		}
		opts.Delay = delay
	}

	if queue := r.URL.Query().Get("queue"); queue != "" {
		if queue != tasks.QueueCritical && queue != tasks.QueueDefault && queue != tasks.QueueLow {
			return nil, fmt.Errorf("invalid queue: %s (must be critical, default, or low)", queue)
		}
		opts.Queue = queue
	}

	if uniqueTTLStr := r.URL.Query().Get("unique_ttl"); uniqueTTLStr != "" {
		uniqueTTL, err := time.ParseDuration(uniqueTTLStr)
		if err != nil {
			return nil, fmt.Errorf("invalid unique_ttl format: %v", err)
		}
		opts.UniqueTTL = uniqueTTL
	}

	if maxRetryStr := r.URL.Query().Get("max_retry"); maxRetryStr != "" {
		maxRetry, err := strconv.Atoi(maxRetryStr)
		if err != nil {
			return nil, fmt.Errorf("invalid max_retry format: %v", err)
		}
		if maxRetry < 0 || maxRetry > 50 {
			return nil, fmt.Errorf("max_retry must be between 0 and 50")
		}
		opts.MaxRetry = maxRetry
	}

	return opts, nil
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := &ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	
	json.NewEncoder(w).Encode(response)
}