// Package main is the single entrypoint for the PUSINGBERAT backend binary.
// It wires dependencies together, sets up the Gin router, and starts the HTTP
// server. No business logic lives here — only wiring and lifecycle management.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api/handler"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/config"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/watcher"
)

func main() {
	// ---------------------------------------------------------------
	// 1. Load configuration from environment variables.
	// ---------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}

	setupLogger(cfg.LogLevel)
	slog.Info("config loaded", "server_addr", cfg.ServerAddress())

	// ---------------------------------------------------------------
	// 2. Set Gin mode based on log level.
	// ---------------------------------------------------------------
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---------------------------------------------------------------
	// 3. Connect to PostgreSQL.
	// ---------------------------------------------------------------
	pool, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}
	defer pool.Close()
	slog.Info("database connected", "host", cfg.DBHost, "db", cfg.DBName)

	// ---------------------------------------------------------------
	// 4. Wire dependency injection:
	//    pgxpool → repos → services → handlers → router
	// ---------------------------------------------------------------

	// Repositories
	logSourceRepo := repository.NewLogSourceRepo(pool)
	eventRepo := repository.NewEventRepo(pool)
	ruleRepo := repository.NewRuleRepo(pool)
	alertRepo := repository.NewAlertRepo(pool)

	// Services
	logSourceSvc := service.NewLogSourceService(logSourceRepo)
	eventSvc := service.NewEventService(eventRepo)
	ruleSvc := service.NewRuleService(ruleRepo)
	alertSvc := service.NewAlertService(alertRepo)

	// Handlers
	logSourceHandler := handler.NewLogSourceHandler(logSourceSvc)
	eventHandler := handler.NewEventHandler(eventSvc)
	ruleHandler := handler.NewRuleHandler(ruleSvc)
	alertHandler := handler.NewAlertHandler(alertSvc)

	// ---------------------------------------------------------------
	// 5. Watcher Pipeline — file watching + event persistence.
	// ---------------------------------------------------------------

	// Create a cancellable context for the entire watcher pipeline.
	// Cancelling watcherCancel stops all file watchers and the
	// persistence worker.
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	defer watcherCancel()

	// Create the watcher registry.
	registry := watcher.NewRegistry(watcherCtx)

	// Attach the registry to the LogSourceService so that Create/Delete
	// automatically start/stop watchers.
	logSourceSvc.SetRegistry(registry)

	// Start the event persistence worker (background goroutine).
	eventSvc.StartPersistenceWorker(watcherCtx, registry.EventChan())

	// Load all existing active log sources and start watching them.
	bootSources, err := logSourceSvc.List(context.Background())
	if err != nil {
		slog.Error("failed to load log sources at boot", "err", err)
	} else {
		registry.StartAll(bootSources)
		slog.Info("watcher pipeline started",
			"sources_loaded", len(bootSources),
			"watchers_active", registry.ActiveCount(),
		)
	}

	// ---------------------------------------------------------------
	// 6. Build the router with all handlers and middleware injected.
	// ---------------------------------------------------------------
	corsOrigins := parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	router := api.NewRouter(api.RouterDeps{
		LogSource:   logSourceHandler,
		Event:       eventHandler,
		Rule:        ruleHandler,
		Alert:       alertHandler,
		Pool:        pool,
		CORSOrigins: corsOrigins,
	})

	// ---------------------------------------------------------------
	// 7. Start the HTTP server with graceful shutdown.
	// ---------------------------------------------------------------
	srv := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("PUSINGBERAT backend starting", "addr", cfg.ServerAddress())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	// ---------------------------------------------------------------
	// 8. Block until OS signal or fatal server error.
	// ---------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received signal, initiating graceful shutdown", "signal", sig)
	case err := <-serverErr:
		log.Fatalf("FATAL: %v", err)
	}

	// ---------------------------------------------------------------
	// 9. Graceful shutdown sequence:
	//    a) Stop all file watchers (cancel watcherCtx)
	//    b) Shutdown HTTP server (drain in-flight requests)
	//    c) Pool.Close() runs via defer
	// ---------------------------------------------------------------
	slog.Info("stopping watcher pipeline")
	watcherCancel()

	// Give in-flight requests up to 10 seconds to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: graceful shutdown failed: %v", err)
	}

	slog.Info("server stopped cleanly")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// connectDB creates a pgxpool connection pool and verifies connectivity.
func connectDB(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// parseCORSOrigins splits a comma-separated list of origins into a slice.
func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return []string{"http://localhost:5173", "http://localhost:5000"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

// setupLogger configures the global slog default based on the log level.
func setupLogger(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}
