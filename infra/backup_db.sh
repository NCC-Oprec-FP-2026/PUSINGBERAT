#!/bin/bash
# ============================================================
# PostgreSQL Automated Backup Script — PUSINGBERAT SIEM
# ============================================================
#
# This script performs a hot backup of the PostgreSQL database
# running inside the Docker container and rotates old backups.
#
# Usage:
#   chmod +x backup_db.sh
#   ./backup_db.sh
#
# Setup Cron (runs daily at 2 AM):
#   0 2 * * * /bin/bash /home/ubuntu/PUSINGBERAT/infra/backup_db.sh >> /home/ubuntu/PUSINGBERAT/infra/backup.log 2>&1
# ============================================================

# --- Configuration ---
CONTAINER_NAME="pusingberat_postgres"
DB_USER="siem"
DB_NAME="pusingberat"
BACKUP_PARENT_DIR="$(dirname "$0")/backups"
BACKUP_RETAIN_DAYS=7

# Ensure backup directory exists
mkdir -p "$BACKUP_PARENT_DIR"

# Timestamp format
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_PARENT_DIR/${DB_NAME}_backup_${TIMESTAMP}.sql.gz"

echo "========================================="
echo "Starting PostgreSQL backup: $(date)"
echo "Target file: $BACKUP_FILE"

# Run pg_dump inside container and compress output on host
if docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
    docker exec -t "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" | gzip > "$BACKUP_FILE"
    
    if [ ${PIPESTATUS[0]} -eq 0 ] && [ ${PIPESTATUS[1]} -eq 0 ]; then
        echo "✅ Backup successfully completed!"
        echo "Size: $(du -sh "$BACKUP_FILE" | cut -f1)"
    else
        echo "❌ Error: pg_dump or gzip failed!"
        rm -f "$BACKUP_FILE"
        exit 1
    fi
else
    echo "❌ Error: Database container '$CONTAINER_NAME' is not running or not ready!"
    exit 1
fi

# --- Retention Policy (Delete backups older than N days) ---
echo "Running retention cleanup (retaining last $BACKUP_RETAIN_DAYS days)..."
find "$BACKUP_PARENT_DIR" -name "${DB_NAME}_backup_*.sql.gz" -mtime +"$BACKUP_RETAIN_DAYS" -exec rm -f {} \; -print
echo "Cleanup finished."
echo "Backup process ended at $(date)"
echo "========================================="
