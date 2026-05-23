package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsHandler struct {
	db *pgxpool.Pool
}

func NewStatsHandler(db *pgxpool.Pool) *StatsHandler {
	return &StatsHandler{db: db}
}

func (h *StatsHandler) RegisterRoutes(group *gin.RouterGroup) {
	stats := group.Group("/stats")
	{
		stats.GET("/overview", h.Overview)
		stats.GET("/events/timeline", h.EventsTimeline)
		stats.GET("/alerts/by-severity", h.AlertsBySeverity)
		stats.GET("/top-sources", h.TopSources)
	}
}

func (h *StatsHandler) Overview(c *gin.Context) {
	var totalEvents24h int64
	var totalAlerts24h int64
	var criticalAlerts int64
	var activeSources int64

	err := h.db.QueryRow(c.Request.Context(), `
		SELECT
			(SELECT COUNT(*) FROM events WHERE received_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(*) FROM alerts WHERE triggered_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(*) FROM alerts WHERE severity = 'critical' AND acknowledged = false),
			(SELECT COUNT(*) FROM log_sources WHERE status = 'active')
	`).Scan(&totalEvents24h, &totalAlerts24h, &criticalAlerts, &activeSources)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_events_24h": totalEvents24h,
			"total_alerts_24h": totalAlerts24h,
			"critical_alerts":  criticalAlerts,
			"active_sources":   activeSources,
		},
	})
}

func (h *StatsHandler) EventsTimeline(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT date_trunc('hour', received_at) AS bucket, COUNT(*) AS count
		FROM events
		WHERE received_at >= NOW() - INTERVAL '24 hours'
		GROUP BY bucket
		ORDER BY bucket ASC
	`)
	if err != nil {
		respondError(c, err)
		return
	}
	defer rows.Close()

	data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (gin.H, error) {
		var bucket time.Time
		var count int64
		if err := row.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		return gin.H{
			"time":  bucket.Format(time.RFC3339),
			"count": count,
		}, nil
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *StatsHandler) AlertsBySeverity(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT severity::text, COUNT(*) AS count
		FROM alerts
		GROUP BY severity
		ORDER BY severity
	`)
	if err != nil {
		respondError(c, err)
		return
	}
	defer rows.Close()

	data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (gin.H, error) {
		var severity string
		var count int64
		if err := row.Scan(&severity, &count); err != nil {
			return nil, err
		}
		return gin.H{
			"severity": severity,
			"count":    count,
		}, nil
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *StatsHandler) TopSources(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT ls.id::text, ls.name, COUNT(e.id) AS event_count
		FROM log_sources ls
		LEFT JOIN events e ON e.log_source_id = ls.id
		GROUP BY ls.id, ls.name
		ORDER BY event_count DESC, ls.name ASC
		LIMIT 5
	`)
	if err != nil {
		respondError(c, err)
		return
	}
	defer rows.Close()

	data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (gin.H, error) {
		var id string
		var name string
		var eventCount int64
		if err := row.Scan(&id, &name, &eventCount); err != nil {
			return nil, err
		}
		return gin.H{
			"id":          id,
			"name":        name,
			"event_count": eventCount,
		}, nil
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
