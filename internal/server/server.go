package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/dysodeng/llm-proxy/internal/config"
	"github.com/dysodeng/llm-proxy/internal/dashboard"
	"github.com/dysodeng/llm-proxy/internal/middleware"
	"github.com/dysodeng/llm-proxy/internal/proxy"
)

// Version is the current server version.
const Version = "1.0.0"

// Server wraps the standard library HTTP server and holds a logger.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// New creates and configures the HTTP server with all routes and middleware.
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	mux := http.NewServeMux()

	// Dashboard
	stats := &dashboard.Stats{}
	baseURL := cfg.Server.ShowBaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}
	dashHandler := dashboard.NewHandler(stats, cfg.RateLimit, Version, baseURL)
	mux.Handle("/", dashHandler)

	// Rate limiter shared across providers
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit)

	// Logging middleware (outermost layer)
	loggingMiddleware := middleware.Logging(logger)

	// OpenAI proxy
	openaiProxy, err := proxy.NewOpenAIProxy(cfg.Providers.OpenAI.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai proxy: %w", err)
	}
	mux.Handle("/openai/", loggingMiddleware(
		rateLimiter.Handler("openai",
			countingHandler("openai", stats, openaiProxy),
		),
	))

	// Anthropic proxy
	anthropicProxy, err := proxy.NewAnthropicProxy(cfg.Providers.Anthropic.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create anthropic proxy: %w", err)
	}
	mux.Handle("/anthropic/", loggingMiddleware(
		rateLimiter.Handler("anthropic",
			countingHandler("anthropic", stats, anthropicProxy),
		),
	))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
	}, nil
}

// Start begins listening and serving HTTP requests. It blocks until the server
// is closed. Returns http.ErrServerClosed on graceful shutdown.
func (s *Server) Start() error {
	s.logger.Info("server starting", zap.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server with a 10-second timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	s.logger.Info("server shutting down")
	return s.httpServer.Shutdown(shutdownCtx)
}

// countingHandler wraps next with a handler that increments request counters in
// stats before delegating to next.
func countingHandler(provider string, stats *dashboard.Stats, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats.Total.Add(1)
		switch provider {
		case "openai":
			stats.OpenAI.Add(1)
		case "anthropic":
			stats.Anthropic.Add(1)
		}
		next.ServeHTTP(w, r)
	})
}
