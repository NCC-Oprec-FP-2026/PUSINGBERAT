// Package main is the single entrypoint for the PUSINGBERAT backend binary.
// It wires dependencies together, sets up the Gin router, and starts the HTTP
// server. No business logic lives here — only wiring and lifecycle management.
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

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/config"
)

func main() {
	// 1. Load configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}

	log.Printf("INFO: config loaded — server will bind on %s", cfg.ServerAddress())

	// 2. Set Gin mode
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 3. Build the router.
	router := buildRouter(cfg)

	// 4. Start the HTTP server with graceful shutdown.
	srv := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in a goroutine so we can listen for OS signals concurrently.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("INFO: PUSINGBERAT backend starting on %s", cfg.ServerAddress())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Block until an OS signal or a fatal server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("INFO: received signal %s, initiating graceful shutdown", sig)
	case err := <-serverErr:
		log.Fatalf("FATAL: %v", err)
	}

	// Give in-flight requests up to 10 seconds to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: graceful shutdown failed: %v", err)
	}

	log.Println("INFO: server stopped cleanly")
}

// buildRouter constructs and returns the Gin engine with all routes registered.
// Day 1 registers only the health endpoint.  Real routes (events, alerts, etc.)
// will be added here on Day 2 once the service layer exists.
func buildRouter(cfg *config.Config) *gin.Engine {
	// gin.New() gives a blank engine without the default Logger/Recovery
	// middleware so we can attach our own structured versions later (Day 2).
	// For Day 1 we attach the built-in ones to keep things working immediately.
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler(cfg))
	}

	// Catch-all for unregistered routes: return a clean JSON 404 rather than
	// Gin's default plain-text response.
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "route not found",
		})
	})

	return router
}

// HealthResponse is the JSON body returned by GET /api/v1/health.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// healthHandler returns a Gin handler that reports the backend's liveness.
// It is intentionally a closure so future sprints can wire in a DB ping check
// without changing the handler's signature.
func healthHandler(_ *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{
			Status:    "ok",
			Service:   "pusingberat-backend",
			Version:   "0.1.0",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}
