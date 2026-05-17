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

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}
	log.Printf("INFO: config loaded, server will bind on %s", cfg.ServerAddress())

	// 2. set gin mode
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 3. build the router
	router := buildRouter(cfg)

	// 4. start the http server with graceful shutdown
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

func buildRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler(cfg))
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "route not found",
		})
	})

	return router
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

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
