#!/bin/sh
set -e

BACKUP_DIR="${BACKUP_DIR:-/backups/redis}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
FILENAME="redis_backup_${TIMESTAMP}.rdb"
MAX_BACKUPS="${MAX_BACKUPS:-30}"

mkdir -p "${BACKUP_DIR}"

if [ -n "$REDIS_PASSWORD" ]; then
  redis-cli -a "$REDIS_PASSWORD" BGSAVE
else
  redis-cli BGSAVE
fi

sleep 2

if [ -f /data/dump.rdb ]; then
  cp /data/dump.rdb "${BACKUP_DIR}/${FILENAME}"
  gzip "${BACKUP_DIR}/${FILENAME}"
  echo "[$(date)] Backup created: ${FILENAME}.gz"
else
  echo "[$(date)] ERROR: dump.rdb not found"
  exit 1
fi

ls -1t "${BACKUP_DIR}"/redis_backup_*.rdb.gz 2>/dev/null | tail -n +${MAX_BACKUPS} | xargs -r rm -
echo "[$(date)] Backup rotation complete (keeping ${MAX_BACKUPS})"
