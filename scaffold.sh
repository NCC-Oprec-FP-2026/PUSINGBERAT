#!/usr/bin/env bash
# =============================================================================
# PUSINGBERAT — Day 1 Scaffolding Script (Revised)
# Assumes you have already cloned the repository and are running this script
# from the root of the cloned repo:
#   git clone https://github.com/NCC-Oprec-FP-2026/PUSINGBERAT.git
#   cd PUSINGBERAT
#   chmod +x scaffold.sh && ./scaffold.sh
# =============================================================================

set -euo pipefail

MODULE_NAME="github.com/NCC-Oprec-FP-2026/PUSINGBERAT"
BACKEND_DIR="backend"

echo "==> Creating backend directory..."
mkdir -p "$BACKEND_DIR"
cd "$BACKEND_DIR"

# -----------------------------------------------------------------------------
# 1. Go module
# -----------------------------------------------------------------------------
echo "==> Initializing Go module: $MODULE_NAME"
go mod init "$MODULE_NAME"

# -----------------------------------------------------------------------------
# 2. Dependencies (exactly as specified in section 16.1 / 4.3 of the doc)
# -----------------------------------------------------------------------------
echo "==> Installing dependencies..."
go get github.com/gin-gonic/gin@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/gorilla/websocket@latest
go get github.com/fsnotify/fsnotify@latest
go get gopkg.in/yaml.v3@latest

echo "==> Tidying module..."
go mod tidy

# -----------------------------------------------------------------------------
# 3. Folder structure (section 2.1)
# -----------------------------------------------------------------------------
echo "==> Creating folder structure..."

# cmd layer
mkdir -p cmd/server

# internal — api layer
mkdir -p internal/api/handler
mkdir -p internal/api/middleware

# internal — remaining packages
mkdir -p internal/config
mkdir -p internal/domain
mkdir -p internal/parser
mkdir -p internal/repository
mkdir -p internal/ruleengine
mkdir -p internal/service
mkdir -p internal/watcher
mkdir -p internal/websocket

# migrations & rules (top-level in backend/)
mkdir -p migrations
mkdir -p rules

# -----------------------------------------------------------------------------
# 4. Placeholder files so `go build ./...` doesn't error on empty packages.
#    These will be replaced by real implementations on Days 2-4.
# -----------------------------------------------------------------------------

# api/handler stubs
for f in alert_handler event_handler logsource_handler rule_handler stats_handler; do
  cat > "internal/api/handler/${f}.go" <<EOF
package handler
EOF
done

# api/middleware stubs
for f in cors logger recovery; do
  cat > "internal/api/middleware/${f}.go" <<EOF
package middleware
EOF
done

# api/router stub
cat > "internal/api/router.go" <<EOF
package api
EOF

# domain stubs
for f in alert event logsource rule; do
  cat > "internal/domain/${f}.go" <<EOF
package domain
EOF
done

# parser stubs
for f in parser syslog_parser nginx_parser generic_parser factory; do
  cat > "internal/parser/${f}.go" <<EOF
package parser
EOF
done

# repository stubs
for f in alert_repo event_repo logsource_repo rule_repo; do
  cat > "internal/repository/${f}.go" <<EOF
package repository
EOF
done

# ruleengine stubs
for f in engine loader matcher; do
  cat > "internal/ruleengine/${f}.go" <<EOF
package ruleengine
EOF
done

# service stubs
for f in alert_service event_service logsource_service rule_service; do
  cat > "internal/service/${f}.go" <<EOF
package service
EOF
done

# watcher stubs
for f in watcher reader registry; do
  cat > "internal/watcher/${f}.go" <<EOF
package watcher
EOF
done

# websocket stubs
for f in hub client; do
  cat > "internal/websocket/${f}.go" <<EOF
package websocket
EOF
done

# -----------------------------------------------------------------------------
# 5. Empty migration files (schema comes from the architecture doc §3.1)
# -----------------------------------------------------------------------------
touch migrations/001_create_tables.sql
touch migrations/002_add_indexes.sql

# -----------------------------------------------------------------------------
# 6. Empty sample rule files
# -----------------------------------------------------------------------------
touch rules/ssh_brute_force.yaml
touch rules/failed_login.yaml
touch rules/high_error_rate.yaml

# -----------------------------------------------------------------------------
# 7. .env.example (section 4.3)
# -----------------------------------------------------------------------------
cat > .env.example <<'EOF'
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=siem
DB_USER=siem
DB_PASSWORD=changeme

# Server
SERVER_PORT=8080

# Alerting
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN

# Rules
RULES_DIR=./rules

# Logging
LOG_LEVEL=info
EOF

echo ""
echo "==> Scaffold complete. Structure:"
find . -type f | sort

echo ""
echo "==> Next steps:"
echo "    1. Copy .env.example to .env and fill in real values."
echo "    2. Replace stub files with the real implementations from the Day 1 deliverables."
echo "    3. Run: go build ./..."
