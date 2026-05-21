// Package main is the single entrypoint for the PUSINGBERAT backend binary.
// It wires dependencies together, sets up the Gin router, and starts the HTTP
// server. No business logic lives here; only wiring and lifecycle management.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api/handler"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/config"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/watcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}

	log.Printf("INFO: config loaded; server will bind on %s", cfg.ServerAddress())

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := pgxpool.New(dbCtx, cfg.DSN())
	if err != nil {
		log.Fatalf("FATAL: connect database: %v\n", err)
	}
	defer db.Close()

	if err := db.Ping(dbCtx); err != nil {
		log.Fatalf("FATAL: ping database: %v\n", err)
	}

	log.Println("INFO: database connection pool ready")

	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := buildRouter(cfg, db)

	srv := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("INFO: PUSINGBERAT backend starting on %s", cfg.ServerAddress())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("INFO: received signal %s, initiating graceful shutdown", sig)
	case err := <-serverErr:
		log.Fatalf("FATAL: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: graceful shutdown failed: %v", err)
	}

	log.Println("INFO: server stopped cleanly")
}

func buildRouter(cfg *config.Config, db *pgxpool.Pool) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	logSourceRepo := repository.NewLogSourceRepo(db)
	eventRepo := repository.NewEventRepo(db)
	ruleRepo := repository.NewRuleRepo(db)
	alertRepo := repository.NewAlertRepo(db)

	eventService := service.NewEventService(eventRepo)
	watcherRegistry := watcher.NewRegistry(eventService)
	logSourceService := service.NewLogSourceService(logSourceRepo, watcherRegistry)
	ruleService := service.NewRuleService(ruleRepo)
	alertService := service.NewAlertService(alertRepo)

	startExistingWatchers(context.Background(), logSourceRepo, watcherRegistry)

	logSourceHandler := handler.NewLogSourceHandler(logSourceService)
	eventHandler := handler.NewEventHandler(eventService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	alertHandler := handler.NewAlertHandler(alertService)
	statsHandler := handler.NewStatsHandler(db)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler(cfg, db))
		logSourceHandler.RegisterRoutes(v1)
		eventHandler.RegisterRoutes(v1)
		ruleHandler.RegisterRoutes(v1)
		alertHandler.RegisterRoutes(v1)
		statsHandler.RegisterRoutes(v1)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "route not found",
		})
	})

	return router
}

func startExistingWatchers(ctx context.Context, repo *repository.LogSourceRepo, registry *watcher.Registry) {
	sources, _, err := repo.List(ctx, repository.LogSourceFilter{
		Status:  domain.LogSourceStatusActive,
		Page:    1,
		PerPage: 200,
	})
	if err != nil {
		log.Printf("WARN: load active log sources failed: %v", err)
		return
	}

	for i := range sources {
		registry.Start(&sources[i])
	}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

func healthHandler(_ *config.Config, db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "error",
				"service":   "pusingberat-backend",
				"database":  "unavailable",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, HealthResponse{
			Status:    "ok",
			Service:   "pusingberat-backend",
			Version:   "0.1.0",
			Database:  "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}
