-- ============================================================
-- PUSINGBERAT SIEM - Initial PostgreSQL schema
-- Creates enum types, core tables, constraints, and updated_at triggers.
-- Indexes used for read in 002_add_indexes.sql.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'severity_level') THEN
        CREATE TYPE severity_level AS ENUM ('info', 'low', 'medium', 'high', 'critical');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'log_source_status') THEN
        CREATE TYPE log_source_status AS ENUM ('active', 'inactive', 'error');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'log_source_type') THEN
        CREATE TYPE log_source_type AS ENUM ('generic', 'syslog', 'nginx');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS log_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    file_path   TEXT NOT NULL UNIQUE,
    log_type    log_source_type NOT NULL DEFAULT 'generic',
    status      log_source_status NOT NULL DEFAULT 'active',
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT log_sources_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT log_sources_file_path_not_blank CHECK (length(btrim(file_path)) > 0)
);

CREATE TABLE IF NOT EXISTS events (
    id            BIGSERIAL PRIMARY KEY,
    log_source_id UUID NOT NULL REFERENCES log_sources(id) ON DELETE CASCADE,
    raw_line      TEXT NOT NULL,
    message       TEXT,
    hostname      VARCHAR(255),
    process       VARCHAR(128),
    pid           INTEGER,
    log_level     VARCHAR(32),
    event_time    TIMESTAMPTZ NOT NULL,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra         JSONB NOT NULL DEFAULT '{}'::jsonb,

    CONSTRAINT events_raw_line_not_blank CHECK (length(btrim(raw_line)) > 0),
    CONSTRAINT events_pid_non_negative CHECK (pid IS NULL OR pid >= 0),
    CONSTRAINT events_extra_is_object CHECK (jsonb_typeof(extra) = 'object')
);

CREATE TABLE IF NOT EXISTS rules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL UNIQUE,
    description  TEXT,
    yaml_content TEXT NOT NULL,
    severity     severity_level NOT NULL DEFAULT 'medium',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT rules_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT rules_yaml_content_not_blank CHECK (length(btrim(yaml_content)) > 0)
);

CREATE TABLE IF NOT EXISTS alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID REFERENCES rules(id) ON DELETE SET NULL,
    rule_name       VARCHAR(255) NOT NULL,
    event_id        BIGINT REFERENCES events(id) ON DELETE SET NULL,
    log_source_id   UUID REFERENCES log_sources(id) ON DELETE SET NULL,
    severity        severity_level NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    raw_line        TEXT,
    triggered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged    BOOLEAN NOT NULL DEFAULT false,
    acknowledged_at TIMESTAMPTZ,
    discord_sent    BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT alerts_rule_name_not_blank CHECK (length(btrim(rule_name)) > 0),
    CONSTRAINT alerts_title_not_blank CHECK (length(btrim(title)) > 0),
    CONSTRAINT alerts_acknowledged_at_when_acknowledged CHECK (
        (acknowledged = false AND acknowledged_at IS NULL)
        OR (acknowledged = true AND acknowledged_at IS NOT NULL)
    )
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- DROP TRIGGER IF EXISTS trg_log_sources_updated_at ON log_sources;
CREATE OR REPLACE TRIGGER trg_log_sources_updated_at
    BEFORE UPDATE ON log_sources
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- DROP TRIGGER IF EXISTS trg_rules_updated_at ON rules;
CREATE OR REPLACE TRIGGER trg_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
