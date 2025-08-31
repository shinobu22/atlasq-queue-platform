package web

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yourorg/atlasq/internal/config"
	"github.com/yourorg/atlasq/internal/logger"
	"github.com/yourorg/atlasq/internal/middleware"
	"github.com/yourorg/atlasq/internal/tasks"
)

type Server struct {
	config         *config.Config
	client         *tasks.Client
	idempotencyMgr *tasks.IdempotencyManager
	server         *http.Server
}

func NewServer(cfg *config.Config, client *tasks.Client, idempotencyMgr *tasks.IdempotencyManager) *Server {
	return &Server{
		config:         cfg,
		client:         client,
		idempotencyMgr: idempotencyMgr,
	}
}

func (s *Server) Start() error {
	router := chi.NewRouter()

	router.Use(middleware.HTTPLogging())

	setupRoutes(router, s.config, s.client, s.idempotencyMgr)

	s.server = &http.Server{
		Addr:    s.config.HTTP.Addr,
		Handler: router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log := logger.Global()
	log.Info("Starting HTTP server",
		zap.String("addr", s.config.HTTP.Addr),
		zap.String("service", s.config.App.Name),
		zap.String("version", s.config.App.Version))

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.Global()
	log.Info("Shutting down HTTP server")

	return s.server.Shutdown(ctx)
}