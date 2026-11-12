#!/usr/bin/env bash
set -Eeuo pipefail

REMOTE_HOST="${REMOTE_HOST:-cbt-vps}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/informatika-monitoring}"
SERVICE_NAME="${SERVICE_NAME:-informatika-monitoring-api.service}"
APP_PORT="${APP_PORT:-8300}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-10}"
HEALTH_RETRY_SECONDS="${HEALTH_RETRY_SECONDS:-2}"
MIN_DISK_MB="${MIN_DISK_MB:-500}"

mode=dispatch
case "${1:-}" in
  --remote) mode=remote ;;
  --check) mode=check ;;
  "") ;;
  *) printf 'Usage: %s [--remote|--check]\n' "$0" >&2; exit 2 ;;
esac

log() { printf '[deploy] %s\n' "$*"; }
fail() { printf '[deploy] ERROR: %s\n' "$*" >&2; exit 1; }
require_command() { command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"; }

if [[ "$mode" == dispatch ]]; then
  [[ "$REMOTE_HOST" =~ ^([A-Za-z0-9][A-Za-z0-9._-]*|[A-Za-z0-9._-]+@[A-Za-z0-9][A-Za-z0-9._-]*)$ ]] || fail "invalid REMOTE_HOST"
  [[ "$REMOTE_DIR" == /* && "$REMOTE_DIR" != *$'\n'* && "$REMOTE_DIR" != *$'\r'* ]] || fail "REMOTE_DIR must be an absolute single-line path"
  printf -v remote_q '%q' "$REMOTE_DIR"
  exec ssh -- "$REMOTE_HOST" "cd -- $remote_q && exec bash deploy/deploy.sh --remote"
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
ENV_FILE="$ROOT_DIR/.env"
RELEASES_DIR="$ROOT_DIR/releases"
BIN_DIR="$ROOT_DIR/bin"
PUBLIC_LINK="$ROOT_DIR/public"
BUILD_ROOT=""
SOURCE_HEAD=""
RELEASE_ID=""
DB_DSN=""
APP_ADDR=""
PUBLIC_DIR=""
OLD_BIN_TARGET=""
OLD_PUBLIC_TARGET=""
MUTATION_STARTED=false
ROLLING_BACK=false

cleanup() {
  if [[ -n "$BUILD_ROOT" && -d "$BUILD_ROOT" ]]; then rm -rf -- "$BUILD_ROOT"; fi
}
trap cleanup EXIT

for value_name in APP_PORT HEALTH_ATTEMPTS HEALTH_RETRY_SECONDS MIN_DISK_MB; do
  value="${!value_name}"
  [[ "$value" =~ ^[0-9]+$ ]] || fail "$value_name must be a non-negative integer"
done
(( APP_PORT >= 1 && APP_PORT <= 65535 )) || fail "APP_PORT must be between 1 and 65535"
(( HEALTH_ATTEMPTS >= 1 && HEALTH_RETRY_SECONDS >= 1 && MIN_DISK_MB >= 1 )) || fail "health and disk values must be positive"
[[ "$SERVICE_NAME" =~ ^[A-Za-z0-9_.@-]+\.service$ ]] || fail "invalid SERVICE_NAME"

parse_environment() {
  [[ -f "$ENV_FILE" && ! -L "$ENV_FILE" ]] || fail ".env must be a regular, non-symlink file"
  local uid mode_bits owner line key raw value
  uid="$(id -u)"; owner="$(stat -c %u "$ENV_FILE")"; mode_bits="$(stat -c %a "$ENV_FILE")"
  [[ "$owner" == "$uid" ]] || fail ".env must be owned by deployment user uid $uid"
  (( (8#$mode_bits & 8#077) == 0 )) || fail ".env must not grant group/other permissions (use chmod 600)"
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^(APP_ADDR|PUBLIC_DIR|DATABASE_URL|GOOSE_DBSTRING)=(.*)$ ]] || fail ".env contains unsupported or malformed assignment"
    key="${BASH_REMATCH[1]}"; raw="${BASH_REMATCH[2]}"
    if [[ "$raw" =~ ^\'([^\']*)\'$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^\"([^\"\\]*)\"$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^[^[:space:]\#\'\"\\]*$ ]]; then value="$raw"
    else fail ".env value for $key is outside the supported literal syntax"
    fi
    case "$key" in
      APP_ADDR) [[ -z "$APP_ADDR" ]] || fail "duplicate APP_ADDR"; APP_ADDR="$value" ;;
      PUBLIC_DIR) [[ -z "$PUBLIC_DIR" ]] || fail "duplicate PUBLIC_DIR"; PUBLIC_DIR="$value" ;;
      DATABASE_URL|GOOSE_DBSTRING)
        [[ -z "$DB_DSN" ]] || fail "define only one of DATABASE_URL or GOOSE_DBSTRING"
        DB_DSN="$value" ;;
    esac
  done < "$ENV_FILE"
  [[ -n "$APP_ADDR" && -n "$PUBLIC_DIR" && -n "$DB_DSN" ]] || fail ".env must define APP_ADDR, PUBLIC_DIR, and DATABASE_URL or GOOSE_DBSTRING"
  [[ "$APP_ADDR" == ":$APP_PORT" ]] || fail "APP_ADDR must be :$APP_PORT"
  [[ "$PUBLIC_DIR" == "$PUBLIC_LINK" ]] || fail "PUBLIC_DIR must be $PUBLIC_LINK"
}

sudo_systemctl() { sudo -n systemctl "$@" "$SERVICE_NAME"; }

check_port_owner() {
  local listeners main_pid cgroup pid
  listeners="$(sudo -n ss -H -ltnp "sport = :$APP_PORT")" || fail "cannot inspect port $APP_PORT with sudo -n ss"
  [[ -n "$listeners" ]] || { log "port $APP_PORT is free"; return; }
  main_pid="$(sudo_systemctl show --property=MainPID --value)" || fail "cannot query service MainPID"
  cgroup="$(sudo_systemctl show --property=ControlGroup --value)" || fail "cannot query service cgroup"
  [[ "$main_pid" =~ ^[1-9][0-9]*$ && -n "$cgroup" && "$cgroup" == /* ]] || fail "port is occupied but service has no verifiable MainPID/cgroup"
  mapfile -t pids < <(printf '%s\n' "$listeners" | sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' | sort -u)
  ((${#pids[@]} > 0)) || fail "port is occupied but listener PID is unavailable"
  for pid in "${pids[@]}"; do
    [[ -r "/proc/$pid/cgroup" ]] || fail "cannot inspect listener PID $pid cgroup"
    grep -Fq -- "$cgroup" "/proc/$pid/cgroup" || fail "port $APP_PORT is owned by PID $pid outside $SERVICE_NAME"
  done
  [[ " ${pids[*]} " == *" $main_pid "* ]] || grep -Fq -- "$cgroup" "/proc/$main_pid/cgroup" || fail "service MainPID cannot be verified"
  log "port $APP_PORT listener belongs to $SERVICE_NAME cgroup"
}

preflight() {
  log "=== PRE-FLIGHT ==="
  [[ "$mode" == check || "$ROOT_DIR" == "$REMOTE_DIR" ]] || fail "remote checkout mismatch: expected $REMOTE_DIR"
  local cmd dirty available_mb root_dev releases_parent_dev
  for cmd in git go node npm sqlc goose psql curl python3 sudo systemctl ss stat df mktemp tar sed sort grep readlink ln mv install timeout id cut cp chmod date; do require_command "$cmd"; done
  parse_environment
  dirty="$(git -C "$ROOT_DIR" status --porcelain)"; [[ -z "$dirty" ]] || fail "working tree is dirty; preserve or commit changes before deployment"
  SOURCE_HEAD="$(git -C "$ROOT_DIR" rev-parse --verify HEAD)"
  git -C "$ROOT_DIR" diff --quiet && git -C "$ROOT_DIR" diff --cached --quiet || fail "working tree is dirty"
  sudo_systemctl show --property=MainPID --value >/dev/null || fail "sudoers must permit systemctl show for $SERVICE_NAME"
  sudo -n ss -H -ltnp "sport = :$APP_PORT" >/dev/null || fail "sudoers must permit ss listener inspection"
  BUILD_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/informatika-deploy.XXXXXX")"
  PGDATABASE="$DB_DSN" timeout 5 psql -X -v ON_ERROR_STOP=1 -Atqc 'SELECT 1' >/dev/null || fail "database connectivity check failed"
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DB_DSN" goose -dir "$ROOT_DIR/backend/sql/migrations" status >/dev/null || fail "goose status failed"
  available_mb="$(df -Pm "$ROOT_DIR" | sed -n '2{s/  */ /g;p}' | cut -d' ' -f4)"
  [[ "$available_mb" =~ ^[0-9]+$ ]] || fail "could not determine free disk space"
  (( available_mb >= MIN_DISK_MB )) || fail "free disk ${available_mb}MB is below ${MIN_DISK_MB}MB"
  root_dev="$(stat -c %d "$ROOT_DIR")"; releases_parent_dev="$root_dev"
  [[ ! -e "$RELEASES_DIR" ]] || releases_parent_dev="$(stat -c %d "$RELEASES_DIR")"
  [[ "$root_dev" == "$releases_parent_dev" ]] || fail "release and stable links must be on the same filesystem"
  check_port_owner
  log "pre-flight passed at $SOURCE_HEAD (secrets redacted)"
}

build_and_verify() {
  log "=== ISOLATED BUILD AND VERIFY ==="
  local src="$BUILD_ROOT/source"
  mkdir -p "$src" "$BUILD_ROOT/npm-cache" "$BUILD_ROOT/go-cache" "$BUILD_ROOT/go-mod-cache"
  git -C "$ROOT_DIR" archive --format=tar "$SOURCE_HEAD" | tar -xf - -C "$src"
  (cd "$src/backend" && GOCACHE="$BUILD_ROOT/go-cache" GOMODCACHE="$BUILD_ROOT/go-mod-cache" sqlc generate && go test ./... -count=1 && go vet ./... && go build -o "$BUILD_ROOT/api" ./cmd/api)
  [[ -x "$BUILD_ROOT/api" ]] || fail "backend binary was not created"
  (cd "$src/frontend" && npm_config_cache="$BUILD_ROOT/npm-cache" npm ci && npm run lint && npm run typecheck && npm run build)
  [[ -f "$src/frontend/out/index.html" ]] || fail "frontend static export is missing"
  log "all gates passed in temporary source pinned to $SOURCE_HEAD"
}

poll_ready() {
  local attempt
  for ((attempt=1; attempt<=HEALTH_ATTEMPTS; attempt++)); do
    curl --fail --silent --show-error --max-time 3 "http://127.0.0.1:$APP_PORT/readyz" >/dev/null && return 0
    log "ready check $attempt/$HEALTH_ATTEMPTS failed"; sleep "$HEALTH_RETRY_SECONDS"
  done
  return 1
}

atomic_link() {
  local target="$1" link="$2" tmp="$link.tmp.$$"
  ln -s -- "$target" "$tmp"
  mv -Tf -- "$tmp" "$link"
}

rollback() {
  local reason="$1" ok=true
  [[ "$ROLLING_BACK" == false ]] || exit 1
  ROLLING_BACK=true; trap - ERR
  log "ROLLBACK: $reason"
  if [[ -n "$OLD_BIN_TARGET" ]]; then atomic_link "$OLD_BIN_TARGET" "$BIN_DIR/current" || ok=false; else rm -f -- "$BIN_DIR/current" || ok=false; fi
  if [[ -n "$OLD_PUBLIC_TARGET" ]]; then atomic_link "$OLD_PUBLIC_TARGET" "$PUBLIC_LINK" || ok=false; else rm -f -- "$PUBLIC_LINK" || ok=false; fi
  sudo_systemctl restart || ok=false
  if [[ "$ok" == true ]]; then poll_ready || ok=false; fi
  if [[ "$ok" == true ]]; then log "rollback restored prior links and verified readiness"; else log "ERROR: rollback did not restore a verified healthy service" >&2; fi
  log "database migrations are forward-only; inspect migration state"
  exit 1
}

on_error() { local rc=$? line=$1; if [[ "$MUTATION_STARTED" == true ]]; then rollback "command failed at line $line (exit $rc)"; fi; exit "$rc"; }
trap 'on_error $LINENO' ERR

deploy_release() {
  local current_head dirty release_dir source_out
  current_head="$(git -C "$ROOT_DIR" rev-parse --verify HEAD)"; [[ "$current_head" == "$SOURCE_HEAD" ]] || fail "source HEAD changed during build"
  dirty="$(git -C "$ROOT_DIR" status --porcelain)"; [[ -z "$dirty" ]] || fail "source checkout became dirty during build"
  RELEASE_ID="$(date -u +%Y%m%dT%H%M%SZ)-${SOURCE_HEAD:0:12}"
  release_dir="$RELEASES_DIR/$RELEASE_ID"
  [[ ! -e "$release_dir" ]] || fail "release already exists: $release_dir"
  source_out="$BUILD_ROOT/source/frontend/out"
  OLD_BIN_TARGET="$(readlink "$BIN_DIR/current" 2>/dev/null || true)"
  OLD_PUBLIC_TARGET="$(readlink "$PUBLIC_LINK" 2>/dev/null || true)"
  [[ ! -e "$BIN_DIR/current" || -L "$BIN_DIR/current" ]] || fail "$BIN_DIR/current must be a symlink"
  [[ ! -e "$PUBLIC_LINK" || -L "$PUBLIC_LINK" ]] || fail "$PUBLIC_LINK must be a symlink"

  MUTATION_STARTED=true
  mkdir -p "$BIN_DIR" "$release_dir/bin" "$release_dir/public"
  install -m 0755 "$BUILD_ROOT/api" "$release_dir/bin/informatika-monitoring-api"
  cp -a "$source_out/." "$release_dir/public/"
  chmod -R a-w "$release_dir"
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DB_DSN" goose -dir "$ROOT_DIR/backend/sql/migrations" up
  atomic_link "$release_dir/bin" "$BIN_DIR/current"
  atomic_link "$release_dir/public" "$PUBLIC_LINK"
  sudo_systemctl restart
  poll_ready
  BASE_URL="http://127.0.0.1:$APP_PORT" "$BUILD_ROOT/source/scripts/smoke.sh"
  MUTATION_STARTED=false
  trap - ERR
  log "deployed immutable release $RELEASE_ID; previous release targets retained"
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DB_DSN" goose -dir "$ROOT_DIR/backend/sql/migrations" version || true
}

preflight
build_and_verify
if [[ "$mode" == check ]]; then
  log "CHECK ONLY: used temporary source/build/cache; no migration, release write, symlink swap, restart, or runtime mutation performed"
  exit 0
fi
deploy_release
