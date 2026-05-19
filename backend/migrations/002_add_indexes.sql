-- ============================================================
-- PUSINGBERAT SIEM — Migration 002: Indexes
-- Run order: MUST run after 001_create_tables.sql
-- ============================================================

-- ============================================================
-- INDEXES: events
-- Primary access patterns:
--   • Dashboard timeline  → filter by event_time range
--   • Event browser       → filter by log_source_id or log_level
--   • Ingestion latency   → sort by received_at
-- ============================================================

-- Most queries filter or sort by event timestamp (dashboard chart, browser pagination)
CREATE INDEX idx_events_event_time
    ON events (event_time DESC);

-- Joining / filtering events by their originating log source
CREATE INDEX idx_events_log_source_id
    ON events (log_source_id);

-- Ordering by ingestion time for latency analysis and "latest events" queries
CREATE INDEX idx_events_received_at
    ON events (received_at DESC);

-- Filtering by severity/level (e.g. "show only ERROR events")
CREATE INDEX idx_events_log_level
    ON events (log_level);

-- GIN index on JSONB extra column — enables fast key/value lookups
-- e.g. WHERE extra @> '{"status_code": "500"}'
CREATE INDEX idx_events_extra_gin
    ON events USING GIN (extra);


-- ============================================================
-- INDEXES: alerts
-- Primary access patterns:
--   • Alert feed         → sort by triggered_at DESC
--   • Severity filter    → filter by severity
--   • Active alert count → WHERE acknowledged = false (partial index)
--   • Source drill-down  → filter by log_source_id
-- ============================================================

-- Default sort order for the alerts table and live feed
CREATE INDEX idx_alerts_triggered_at
    ON alerts (triggered_at DESC);

-- Severity filter on the alerts page
CREATE INDEX idx_alerts_severity
    ON alerts (severity);

-- Partial index — covers ONLY unacknowledged alerts.
-- Makes "active alert count" widget extremely fast regardless of total alert volume.
CREATE INDEX idx_alerts_unack
    ON alerts (acknowledged)
    WHERE acknowledged = false;

-- Filter alerts by originating log source
CREATE INDEX idx_alerts_log_source_id
    ON alerts (log_source_id);