-- ============================================================
-- PUSINGBERAT SIEM - Query and dashboard indexes
-- ============================================================

-- Log sources: common list and health filters.
CREATE INDEX IF NOT EXISTS idx_log_sources_status ON log_sources (status);
CREATE INDEX IF NOT EXISTS idx_log_sources_log_type ON log_sources (log_type);
CREATE INDEX IF NOT EXISTS idx_log_sources_created_at ON log_sources (created_at DESC);

-- Events: high-volume table queried by timeline, source, level, and recency.
CREATE INDEX IF NOT EXISTS idx_events_event_time ON events (event_time DESC);
CREATE INDEX IF NOT EXISTS idx_events_received_at ON events (received_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_log_source_id ON events (log_source_id);
CREATE INDEX IF NOT EXISTS idx_events_log_source_event_time ON events (log_source_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_events_log_level ON events (log_level);
CREATE INDEX IF NOT EXISTS idx_events_extra_gin ON events USING GIN (extra);

-- Rules: management UI lists enabled rules and filters by severity.
CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules (enabled);
CREATE INDEX IF NOT EXISTS idx_rules_severity ON rules (severity);
CREATE INDEX IF NOT EXISTS idx_rules_updated_at ON rules (updated_at DESC);

-- Alerts: dashboard, filters, active alert widget, and webhook retry flow.
CREATE INDEX IF NOT EXISTS idx_alerts_triggered_at ON alerts (triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts (severity);
CREATE INDEX IF NOT EXISTS idx_alerts_log_source_id ON alerts (log_source_id);
CREATE INDEX IF NOT EXISTS idx_alerts_rule_id ON alerts (rule_id);
CREATE INDEX IF NOT EXISTS idx_alerts_event_id ON alerts (event_id);
CREATE INDEX IF NOT EXISTS idx_alerts_unacknowledged ON alerts (triggered_at DESC)
WHERE acknowledged = false;
CREATE INDEX IF NOT EXISTS idx_alerts_discord_pending ON alerts (triggered_at ASC)
WHERE discord_sent = false;
CREATE INDEX IF NOT EXISTS idx_alerts_severity_triggered_at ON alerts (severity, triggered_at DESC);
