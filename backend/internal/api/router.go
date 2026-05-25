package api

import (
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api/handler"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api/middleware"
	ws "github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/websocket"
)

// RouterDeps bundles all handler instances that the router needs.
// This avoids a long constructor parameter list and makes it easy to
// add new handlers in future sprints.
type RouterDeps struct {
	LogSource *handler.LogSourceHandler
	Event     *handler.EventHandler
	Rule      *handler.RuleHandler
	Alert     *handler.AlertHandler
	Stats     *handler.StatsHandler

	// Pool is kept so the health endpoint can run a DB ping check.
	Pool *pgxpool.Pool

	// WSHub is the WebSocket hub for real-time alert broadcasting.
	WSHub *ws.Hub

	// CORSOrigins is a list of allowed frontend origins for the CORS
	// middleware (read from CORS_ALLOWED_ORIGINS env var).
	CORSOrigins []string
}

// NewRouter constructs the Gin engine with all middleware and routes
// registered. This replaces the Day 1 buildRouter() function in main.go.
func NewRouter(deps RouterDeps) *gin.Engine {
	router := gin.New()

	// Global middleware stack (order matters).
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: deps.CORSOrigins,
	}))

	// API v1 group.
	v1 := router.Group("/api/v1")
	{
		// Health check (includes DB ping).
		v1.GET("/health", healthHandler(deps.Pool))

		// Log Sources — full CRUD.
		sources := v1.Group("/sources")
		{
			sources.GET("", deps.LogSource.List)
			sources.POST("", deps.LogSource.Create)
			sources.GET("/:id", deps.LogSource.GetByID)
			sources.PATCH("/:id", deps.LogSource.Update)
			sources.DELETE("/:id", deps.LogSource.Delete)
		}

		// Events — read-only.
		events := v1.Group("/events")
		{
			events.GET("", deps.Event.List)
			events.GET("/:id", deps.Event.GetByID)
		}

		// Rules — full CRUD + toggle.
		rules := v1.Group("/rules")
		{
			rules.GET("", deps.Rule.List)
			rules.POST("", deps.Rule.Create)
			rules.GET("/:id", deps.Rule.GetByID)
			rules.PUT("/:id", deps.Rule.Update)
			rules.PATCH("/:id/toggle", deps.Rule.Toggle)
			rules.DELETE("/:id", deps.Rule.Delete)
		}

		// Alerts — read + acknowledge + delete.
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", deps.Alert.List)
			alerts.GET("/severitycount", deps.Alert.GetSeverityCounts)
			alerts.GET("/:id", deps.Alert.GetByID)
			alerts.PATCH("/:id/acknowledge", deps.Alert.Acknowledge)
			alerts.DELETE("/:id", deps.Alert.Delete)
		}

		// Stats — dashboard dashboard endpoint routes.
		stats := v1.Group("/stats")
		{
			stats.GET("/overview", deps.Stats.Overview)
			stats.GET("/events/timeline", deps.Stats.EventsTimeline)
			stats.GET("/alerts/by-severity", deps.Stats.AlertsBySeverity)
			stats.GET("/top-sources", deps.Stats.TopSources)
		}
	}

	// ---------------------------------------------------------------
	// pprof - Performance profiling endpoints (enabled via env flag)
	// ---------------------------------------------------------------
	if os.Getenv("ENABLE_PPROF") == "true" {
		debug := router.Group("/debug/pprof")
		{
			debug.GET("/", gin.WrapF(pprof.Index))
			debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
			debug.GET("/profile", gin.WrapF(pprof.Profile))
			debug.POST("/symbol", gin.WrapF(pprof.Symbol))
			debug.GET("/symbol", gin.WrapF(pprof.Symbol))
			debug.GET("/trace", gin.WrapF(pprof.Trace))
			debug.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
			debug.GET("/block", gin.WrapH(pprof.Handler("block")))
			debug.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
			debug.GET("/heap", gin.WrapH(pprof.Handler("heap")))
			debug.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
			debug.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		}
	}

	// ---------------------------------------------------------------
	// WebSocket endpoint — real-time alert streaming.
	// ---------------------------------------------------------------
	if deps.WSHub != nil {
		router.GET("/ws", func(c *gin.Context) {
			ws.ServeWS(deps.WSHub, c.Writer, c.Request)
		})
	}

	// Catch-all: clean JSON 404 for unregistered routes.
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "route not found",
		})
	})

	return router
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	DBStatus  string `json:"db_status"`
}

// healthHandler returns a handler that reports backend liveness with a DB ping.
func healthHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if err := pool.Ping(c.Request.Context()); err != nil {
			dbStatus = "unreachable"
		}

		status := "ok"
		if dbStatus != "ok" {
			status = "degraded"
		}

		c.JSON(http.StatusOK, healthResponse{
			Status:    status,
			Service:   "pusingberat-backend",
			Version:   "0.2.0",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			DBStatus:  dbStatus,
		})
	}
}
