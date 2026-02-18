#!/usr/bin/env bash
set -euo pipefail

DB_CONTAINER="${DB_CONTAINER:-cbtlms-postgres}"
DB_USER="${DB_USER:-cbtlms}"
DB_NAME="${DB_NAME:-cbtlms}"
BACKUP_DIR="${BACKUP_DIR:-/tmp/cbtlms-backups}"
COMPRESS="${COMPRESS:-true}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

usage() {
  cat <<USAGE
Usage: scripts/backup.sh

Env overrides:
  DB_CONTAINER    (default: cbtlms-postgres)
  DB_USER         (default: cbtlms)
  DB_NAME         (default: cbtlms)
  BACKUP_DIR      (default: /tmp/cbtlms-backups)
  COMPRESS        (default: true)  -> true/false
  RETENTION_DAYS  (default: 7)     -> old backups removed automatically
USAGE
}

require_tools() {
  command -v docker >/dev/null 2>&1 || { echo "docker not found"; exit 1; }
  command -v sha256sum >/dev/null 2>&1 || { echo "sha256sum not found"; exit 1; }
}

check_container() {
  if ! docker ps --format '{{.Names}}' | grep -qx "$DB_CONTAINER"; then
    echo "Container '$DB_CONTAINER' is not running."
    echo "Start it: docker compose -f deployments/docker-compose.yml up -d postgres"
    exit 1
  fi
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "${1:-}" == "help" ]]; then
    usage
    exit 0
  fi

  require_tools
  check_container

  mkdir -p "$BACKUP_DIR"

  ts="$(date +%Y%m%d_%H%M%S)"
  base="cbtlms_${DB_NAME}_${ts}.sql"
  out_file="$BACKUP_DIR/$base"

  echo "Creating backup: $out_file"
  docker exec "$DB_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" > "$out_file"

  if [[ "$COMPRESS" == "true" ]]; then
    gzip -f "$out_file"
    out_file="$out_file.gz"
  fi

  sha256sum "$out_file" > "$out_file.sha256"
  echo "Checksum: $out_file.sha256"

  if [[ "$RETENTION_DAYS" =~ ^[0-9]+$ ]] && [[ "$RETENTION_DAYS" -gt 0 ]]; then
    find "$BACKUP_DIR" -type f \( -name '*.sql' -o -name '*.sql.gz' -o -name '*.sha256' \) -mtime +"$RETENTION_DAYS" -print -delete || true
  fi

  echo "Backup success: $out_file"
}

main "$@"
