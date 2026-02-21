#!/usr/bin/env bash
set -euo pipefail

DB_CONTAINER="${DB_CONTAINER:-cbtlms-postgres}"
DB_USER="${DB_USER:-cbtlms}"
DB_NAME="${DB_NAME:-cbtlms}"

usage() {
  cat <<USAGE
Usage:
  scripts/restore_check.sh <backup_file.sql|backup_file.sql.gz|backup_file.sql.enc|backup_file.sql.gz.enc>

This script is non-destructive to main DB:
- creates temporary database
- restores backup into temp DB
- runs sanity checks
- drops temp DB

Env overrides:
  DB_CONTAINER (default: cbtlms-postgres)
  DB_USER      (default: cbtlms)
  DB_NAME      (default: cbtlms)
  BACKUP_PASSPHRASE (required for *.enc backup files)
USAGE
}

require_tools() {
  command -v docker >/dev/null 2>&1 || { echo "docker not found"; exit 1; }
  command -v gunzip >/dev/null 2>&1 || { echo "gunzip not found"; exit 1; }
  command -v openssl >/dev/null 2>&1 || { echo "openssl not found"; exit 1; }
}

check_container() {
  if ! docker ps --format '{{.Names}}' | grep -qx "$DB_CONTAINER"; then
    echo "Container '$DB_CONTAINER' is not running."
    echo "Start it: docker compose -f deployments/docker-compose.yml up -d postgres"
    exit 1
  fi
}

main() {
  if [[ $# -lt 1 ]]; then
    usage
    exit 1
  fi

  backup_file="$1"
  if [[ ! -f "$backup_file" ]]; then
    echo "Backup file not found: $backup_file"
    exit 1
  fi

  require_tools
  check_container

  temp_db="${DB_NAME}_restore_check_$(date +%s)"

  cleanup() {
    docker exec "$DB_CONTAINER" dropdb -U "$DB_USER" --if-exists "$temp_db" >/dev/null 2>&1 || true
  }
  trap cleanup EXIT

  echo "Creating temp DB: $temp_db"
  docker exec "$DB_CONTAINER" createdb -U "$DB_USER" "$temp_db"

  echo "Restoring backup into temp DB"
  if [[ "$backup_file" == *.gz.enc ]]; then
    if [[ -z "${BACKUP_PASSPHRASE:-}" ]]; then
      echo "BACKUP_PASSPHRASE is required for encrypted backup file"
      exit 1
    fi
    openssl enc -d -aes-256-cbc -pbkdf2 -in "$backup_file" -pass "env:BACKUP_PASSPHRASE" \
      | gunzip -c \
      | docker exec -i "$DB_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$temp_db" >/dev/null
  elif [[ "$backup_file" == *.sql.enc ]]; then
    if [[ -z "${BACKUP_PASSPHRASE:-}" ]]; then
      echo "BACKUP_PASSPHRASE is required for encrypted backup file"
      exit 1
    fi
    openssl enc -d -aes-256-cbc -pbkdf2 -in "$backup_file" -pass "env:BACKUP_PASSPHRASE" \
      | docker exec -i "$DB_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$temp_db" >/dev/null
  elif [[ "$backup_file" == *.gz ]]; then
    gunzip -c "$backup_file" | docker exec -i "$DB_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$temp_db" >/dev/null
  else
    cat "$backup_file" | docker exec -i "$DB_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$temp_db" >/dev/null
  fi

  echo "Running sanity checks"
  table_count="$(docker exec "$DB_CONTAINER" psql -At -U "$DB_USER" -d "$temp_db" -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")"
  if [[ "$table_count" -lt 5 ]]; then
    echo "Sanity check failed: expected >=5 public tables, got $table_count"
    exit 1
  fi

  if docker exec "$DB_CONTAINER" psql -At -U "$DB_USER" -d "$temp_db" -c "SELECT to_regclass('public.schema_migrations') IS NOT NULL;" | grep -qx 't'; then
    migration_count="$(docker exec "$DB_CONTAINER" psql -At -U "$DB_USER" -d "$temp_db" -c "SELECT COUNT(*) FROM schema_migrations;")"
    echo "schema_migrations rows: $migration_count"
  fi

  echo "Restore check success on temp DB: $temp_db"
}

main "$@"
