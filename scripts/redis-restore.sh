#!/bin/sh
set -e

BACKUP_DIR="${BACKUP_DIR:-/backups/redis}"
RESTORE_FILE="${RESTORE_FILE:-}"

if [ -z "$RESTORE_FILE" ]; then
  echo "ERROR: RESTORE_FILE environment variable is required"
  echo "Usage: RESTORE_FILE=redis_backup_20240101_120000.rdb.gz ./redis-restore.sh"
  exit 1
fi

if [ ! -f "${BACKUP_DIR}/${RESTORE_FILE}" ]; then
  echo "ERROR: Backup file not found: ${BACKUP_DIR}/${RESTORE_FILE}"
  exit 1
fi

echo "[$(date)] Stopping Redis..."
if [ -n "$REDIS_PASSWORD" ]; then
  redis-cli -a "$REDIS_PASSWORD" SHUTDOWN NOSAVE || true
else
  redis-cli SHUTDOWN NOSAVE || true
fi

sleep 2

echo "[$(date)] Restoring backup: ${RESTORE_FILE}"
gunzip -c "${BACKUP_DIR}/${RESTORE_FILE}" > /data/dump.rdb

echo "[$(date)] Starting Redis..."
redis-server --daemonize yes --requirepass "$REDIS_PASSWORD"

sleep 2

if [ -n "$REDIS_PASSWORD" ]; then
  redis-cli -a "$REDIS_PASSWORD" PING
else
  redis-cli PING
fi

echo "[$(date)] Restore complete"
