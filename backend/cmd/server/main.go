// Package main is the single entrypoint for the PUSINGBERAT backend binary.
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
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/notifier"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/watcher"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}

	setupLogger(cfg.LogLevel)
	slog.Info("config loaded", "server_addr", cfg.ServerAddress())

	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	pool, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}
	defer pool.Close()
	slog.Info("database connected", "host", cfg.DBHost, "db", cfg.DBName)

	logSourceRepo := repository.NewLogSourceRepo(pool)
	eventRepo := repository.NewEventRepo(pool)
	ruleRepo := repository.NewRuleRepo(pool)
	alertRepo := repository.NewAlertRepo(pool)

	if err := ruleengine.SeedRules(context.Background(), ruleRepo, cfg.RulesDir); err != nil {
		slog.Warn("seed rules failed", "err", err)
	}
	loadedRules, err := ruleengine.LoadEnabledRulesFromDB(context.Background(), ruleRepo)
	if err != nil {
		slog.Warn("load enabled rules failed", "err", err)
	}
	slog.Info("rule engine loaded", "rules", len(loadedRules))

	alertChan := make(chan domain.Alert, 100)
	wsAlertChan := make(chan domain.Alert, 100)
	engine := ruleengine.NewEngine(loadedRules)
	alertSvc := service.NewAlertService(alertRepo)

	var discordNotifier ruleengine.AlertNotifier
	if cfg.DiscordWebhookURL != "" {
		discordNotifier = notifier.NewDiscordNotifier(cfg.DiscordWebhookURL)
		slog.Info("discord notifier enabled")
	} else {
		slog.Info("discord notifier disabled", "reason", "DISCORD_WEBHOOK_URL is empty")
	}

	hub := websocket.NewHub()
	go hub.Run(context.Background())
	go forwardAlertsToWebsocket(context.Background(), wsAlertChan, hub)

	dispatcher := ruleengine.NewAlertDispatcher(alertChan, alertSvc, discordNotifier, wsAlertChan)
	go dispatcher.Run(context.Background())

	logSourceSvc := service.NewLogSourceService(logSourceRepo)
	eventSvc := service.NewEventService(eventRepo, engine, alertChan)
	ruleSvc := service.NewRuleService(ruleRepo)

	logSourceHandler := handler.NewLogSourceHandler(logSourceSvc)
	eventHandler := handler.NewEventHandler(eventSvc)
	ruleHandler := handler.NewRuleHandler(ruleSvc)
	alertHandler := handler.NewAlertHandler(alertSvc)

	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	defer watcherCancel()

	registry := watcher.NewRegistry(watcherCtx)
	logSourceSvc.SetRegistry(registry)
	eventSvc.StartPersistenceWorker(watcherCtx, registry.EventChan())

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

	router := api.NewRouter(api.RouterDeps{
		LogSource:   logSourceHandler,
		Event:       eventHandler,
		Rule:        ruleHandler,
		Alert:       alertHandler,
		Pool:        pool,
		CORSOrigins: parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
	})
	router.GET("/ws", hub.Handle)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received signal, initiating graceful shutdown", "signal", sig)
	case err := <-serverErr:
		log.Fatalf("FATAL: %v", err)
	}

	slog.Info("stopping watcher pipeline")
	watcherCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: graceful shutdown failed: %v", err)
	}

	slog.Info("server stopped cleanly")
}

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

func forwardAlertsToWebsocket(ctx context.Context, alerts <-chan domain.Alert, hub *websocket.Hub) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert, ok := <-alerts:
			if !ok {
				return
			}
			hub.BroadcastAlert(alert)
		}
	}
}

func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return []string{"http://localhost:5173", "http://localhost:5000"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

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
