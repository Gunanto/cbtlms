#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-status}"

CONTAINER_NAME="${DB_CONTAINER:-cbtlms-postgres}"
DB_USER="${DB_USER:-cbtlms}"
DB_NAME="${DB_NAME:-cbtlms}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-internal/db/migrations}"
MIGRATION_TABLE="${MIGRATION_TABLE:-schema_migrations}"

usage() {
  cat <<USAGE
Usage:
  scripts/migrate.sh up [N]
  scripts/migrate.sh down [N]
  scripts/migrate.sh status
  scripts/migrate.sh reset

Env overrides:
  DB_CONTAINER   (default: cbtlms-postgres)
  DB_USER        (default: cbtlms)
  DB_NAME        (default: cbtlms)
  MIGRATIONS_DIR (default: internal/db/migrations)
  MIGRATION_TABLE (default: schema_migrations)
USAGE
}

require_tools() {
  command -v docker >/dev/null 2>&1 || { echo "docker not found"; exit 1; }
  command -v sort >/dev/null 2>&1 || { echo "sort not found"; exit 1; }
}

check_container() {
  if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    echo "Container '$CONTAINER_NAME' is not running."
    echo "Start it: docker compose -f deployments/docker-compose.yml up -d postgres"
    exit 1
  fi
}

psql_exec() {
  local sql="$1"
  docker exec "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" -c "$sql" >/dev/null
}

psql_query() {
  local sql="$1"
  docker exec "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -At -U "$DB_USER" -d "$DB_NAME" -c "$sql"
}

ensure_migration_table() {
  local exists
  exists="$(psql_query "SELECT to_regclass('public.${MIGRATION_TABLE}') IS NOT NULL;")"
  if [[ "$exists" == "t" ]]; then
    return
  fi

  psql_exec "
    CREATE TABLE ${MIGRATION_TABLE} (
      version VARCHAR(32) PRIMARY KEY,
      name TEXT NOT NULL,
      applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );
  "
}

list_up_files() {
  find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.up.sql' | sort
}

version_from_file() {
  local file="$1"
  local base
  base="$(basename "$file")"
  echo "${base%%_*}"
}

name_from_file() {
  local file="$1"
  basename "$file"
}

is_applied() {
  local version="$1"
  local out
  out="$(psql_query "SELECT 1 FROM ${MIGRATION_TABLE} WHERE version='${version}' LIMIT 1;")"
  [[ "$out" == "1" ]]
}

record_applied() {
  local version="$1"
  local name="$2"
  psql_exec "
    INSERT INTO ${MIGRATION_TABLE}(version, name)
    VALUES ('${version}', '${name}')
    ON CONFLICT (version) DO NOTHING;
  "
}

remove_record() {
  local version="$1"
  psql_exec "DELETE FROM ${MIGRATION_TABLE} WHERE version='${version}';"
}

run_sql_file() {
  local file="$1"
  docker exec -i "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" < "$file"
}

apply_up() {
  local limit="${1:-0}"
  local applied=0

  while IFS= read -r file; do
    local version name
    version="$(version_from_file "$file")"
    name="$(name_from_file "$file")"

    if is_applied "$version"; then
      echo "SKIP  $name (already applied)"
      continue
    fi

    echo "APPLY $name"
    output=""
    if output="$(run_sql_file "$file" 2>&1)"; then
      record_applied "$version" "$name"
    else
      if echo "$output" | grep -Eiq "already exists|duplicate key value|constraint .* already exists|relation .* already exists"; then
        echo "ADOPT $name (schema already present)"
        record_applied "$version" "$name"
      else
        echo "$output"
        echo "ERROR applying $name"
        exit 1
      fi
    fi
    applied=$((applied + 1))

    if [ "$limit" -gt 0 ] && [ "$applied" -ge "$limit" ]; then
      break
    fi
  done < <(list_up_files)

  if [ "$applied" -eq 0 ]; then
    echo "No new migrations to apply"
  fi
}

apply_down() {
  local limit="${1:-1}"
  local reverted=0

  while IFS='|' read -r version name; do
    [ -z "$version" ] && continue

    local down_file
    down_file="${MIGRATIONS_DIR}/${name%.up.sql}.down.sql"

    if [ ! -f "$down_file" ]; then
      # fallback by searching any matching version down file
      down_file="$(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name "${version}_*.down.sql" | sort | head -n1 || true)"
    fi

    if [ -z "$down_file" ] || [ ! -f "$down_file" ]; then
      echo "ERROR: down migration file not found for version $version"
      exit 1
    fi

    echo "REVERT $(basename "$down_file")"
    run_sql_file "$down_file"
    remove_record "$version"
    reverted=$((reverted + 1))

    if [ "$limit" -gt 0 ] && [ "$reverted" -ge "$limit" ]; then
      break
    fi
  done < <(psql_query "SELECT version, name FROM ${MIGRATION_TABLE} ORDER BY version DESC;")

  if [ "$reverted" -eq 0 ]; then
    echo "No applied migrations to revert"
  fi
}

print_status() {
  echo "Container : $CONTAINER_NAME"
  echo "Database  : $DB_NAME"
  echo "User      : $DB_USER"
  echo "Migrations dir: $MIGRATIONS_DIR"
  echo

  echo "Applied:"
  psql_query "SELECT version || ' ' || name || ' @ ' || applied_at FROM ${MIGRATION_TABLE} ORDER BY version;" || true
  echo

  echo "Pending:"
  local pending=0
  while IFS= read -r file; do
    local version name
    version="$(version_from_file "$file")"
    name="$(name_from_file "$file")"
    if ! is_applied "$version"; then
      echo "$version $name"
      pending=$((pending + 1))
    fi
  done < <(list_up_files)

  if [ "$pending" -eq 0 ]; then
    echo "(none)"
  fi
}

require_tools
check_container
ensure_migration_table

case "$ACTION" in
  up)
    apply_up "${2:-0}"
    ;;
  down)
    apply_down "${2:-1}"
    ;;
  reset)
    apply_down 0
    apply_up 0
    ;;
  status)
    print_status
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unknown action: $ACTION"
    usage
    exit 1
    ;;
esac
