#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
COOKIE_JAR="${COOKIE_JAR:-cookie.txt}"

usage() {
  cat <<USAGE
Usage:
  scripts/csrf-curl.sh login <identifier> <password>
  scripts/csrf-curl.sh otp-login <email> <code>
  scripts/csrf-curl.sh token
  scripts/csrf-curl.sh req <METHOD> <PATH> [curl_args...]

Examples:
  scripts/csrf-curl.sh login admin Admin123!
  scripts/csrf-curl.sh token
  scripts/csrf-curl.sh req GET /api/v1/auth/me
  scripts/csrf-curl.sh req POST /api/v1/auth/logout
  scripts/csrf-curl.sh req POST /api/v1/attempts/1/submit

Env overrides:
  BASE_URL   (default: http://localhost:8080)
  COOKIE_JAR (default: cookie.txt)
USAGE
}

csrf_token() {
  if [[ ! -f "$COOKIE_JAR" ]]; then
    return 1
  fi
  awk '$6=="cbtlms_csrf"{print $7}' "$COOKIE_JAR" | tail -n1
}

require_token() {
  local token
  token="$(csrf_token || true)"
  if [[ -z "$token" ]]; then
    echo "CSRF token not found. Run login first." >&2
    exit 1
  fi
  printf '%s' "$token"
}

cmd_login() {
  local identifier="${1:-}"
  local password="${2:-}"
  if [[ -z "$identifier" || -z "$password" ]]; then
    echo "login requires: <identifier> <password>" >&2
    exit 1
  fi

  curl -sS -i -c "$COOKIE_JAR" -X POST "$BASE_URL/api/v1/auth/login-password" \
    -H 'Content-Type: application/json' \
    -d "{\"identifier\":\"$identifier\",\"password\":\"$password\"}"

  echo
  echo "Saved cookie jar: $COOKIE_JAR"
  echo "CSRF token: $(csrf_token || echo '<not found>')"
}

cmd_otp_login() {
  local email="${1:-}"
  local code="${2:-}"
  if [[ -z "$email" || -z "$code" ]]; then
    echo "otp-login requires: <email> <code>" >&2
    exit 1
  fi

  curl -sS -i -c "$COOKIE_JAR" -X POST "$BASE_URL/api/v1/auth/otp/verify" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"code\":\"$code\"}"

  echo
  echo "Saved cookie jar: $COOKIE_JAR"
  echo "CSRF token: $(csrf_token || echo '<not found>')"
}

cmd_token() {
  local token
  token="$(csrf_token || true)"
  if [[ -z "$token" ]]; then
    echo "<empty>"
    exit 1
  fi
  echo "$token"
}

needs_csrf() {
  local method="$1"
  case "$method" in
    POST|PUT|PATCH|DELETE) return 0 ;;
    *) return 1 ;;
  esac
}

cmd_req() {
  local method="${1:-}"
  local path="${2:-}"
  shift 2 || true

  if [[ -z "$method" || -z "$path" ]]; then
    echo "req requires: <METHOD> <PATH>" >&2
    exit 1
  fi

  method="$(echo "$method" | tr '[:lower:]' '[:upper:]')"
  local url
  url="$BASE_URL$path"

  if needs_csrf "$method"; then
    local token
    token="$(require_token)"
    curl -sS -i -b "$COOKIE_JAR" -X "$method" "$url" \
      -H "X-CSRF-Token: $token" \
      "$@"
    return
  fi

  curl -sS -i -b "$COOKIE_JAR" -X "$method" "$url" "$@"
}

main() {
  local cmd="${1:-}"
  shift || true

  case "$cmd" in
    login)
      cmd_login "$@"
      ;;
    otp-login)
      cmd_otp_login "$@"
      ;;
    token)
      cmd_token
      ;;
    req)
      cmd_req "$@"
      ;;
    -h|--help|help|"")
      usage
      ;;
    *)
      echo "Unknown command: $cmd" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
