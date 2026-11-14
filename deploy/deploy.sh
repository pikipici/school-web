#!/usr/bin/env bash
set -Eeuo pipefail

REMOTE_HOST="${REMOTE_HOST:-cbt-vps}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/informatika-monitoring}"
SERVICE_NAME="${SERVICE_NAME:-informatika-monitoring-api.service}"
APP_PORT="${APP_PORT:-8300}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-10}"
HEALTH_RETRY_SECONDS="${HEALTH_RETRY_SECONDS:-2}"
MIN_DISK_MB="${MIN_DISK_MB:-500}"
BACKUPS_TO_KEEP="${BACKUPS_TO_KEEP:-10}"
TEST_ENV_FILE="${TEST_ENV_FILE:-${HOME:-/nonexistent}/.informatika-monitoring-test.env}"

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
TEST_DB_DSN=""
BACKUP_FILE=""
ATTACHMENTS_SNAPSHOT=""
BACKUP_DIR="$ROOT_DIR/backups"
ADMIN_COOKIE_SECURE=""
CSP_MODE=""
ATTACHMENTS_DIR=""
ATTACHMENTS_SEEN=false
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

for value_name in APP_PORT HEALTH_ATTEMPTS HEALTH_RETRY_SECONDS MIN_DISK_MB BACKUPS_TO_KEEP; do
  value="${!value_name}"
  [[ "$value" =~ ^[0-9]+$ ]] || fail "$value_name must be a non-negative integer"
done
(( APP_PORT >= 1 && APP_PORT <= 65535 )) || fail "APP_PORT must be between 1 and 65535"
(( HEALTH_ATTEMPTS >= 1 && HEALTH_RETRY_SECONDS >= 1 && MIN_DISK_MB >= 1 && BACKUPS_TO_KEEP >= 1 )) || fail "health, disk, and backup values must be positive"
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
    [[ "$line" =~ ^(APP_ADDR|PUBLIC_DIR|DATABASE_URL|GOOSE_DBSTRING|ADMIN_COOKIE_SECURE|CSP_MODE|ATTACHMENTS_DIR)=(.*)$ ]] || fail ".env contains unsupported or malformed assignment"
    key="${BASH_REMATCH[1]}"; raw="${BASH_REMATCH[2]}"
    if [[ "$raw" =~ ^\'([^\']*)\'$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^\"([^\"\\]*)\"$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^[^[:space:]\#\'\"\\]*$ ]]; then value="$raw"
    else fail ".env value for $key is outside the supported literal syntax"
    fi
    case "$key" in
      APP_ADDR) [[ -z "$APP_ADDR" ]] || fail "duplicate APP_ADDR"; APP_ADDR="$value" ;;
      PUBLIC_DIR) [[ -z "$PUBLIC_DIR" ]] || fail "duplicate PUBLIC_DIR"; PUBLIC_DIR="$value" ;;
      ADMIN_COOKIE_SECURE)
        [[ -z "$ADMIN_COOKIE_SECURE" ]] || fail "duplicate ADMIN_COOKIE_SECURE"
        [[ "${value,,}" =~ ^(true|false|1|0)$ ]] || fail "ADMIN_COOKIE_SECURE must be true, false, 1, or 0"
        ADMIN_COOKIE_SECURE="${value,,}" ;;
      CSP_MODE)
        [[ -z "$CSP_MODE" ]] || fail "duplicate CSP_MODE"
        [[ "${value,,}" =~ ^(enforce|report-only|off)$ ]] || fail "CSP_MODE must be enforce, report-only, or off"
        CSP_MODE="${value,,}" ;;
      ATTACHMENTS_DIR)
        [[ "$ATTACHMENTS_SEEN" == false ]] || fail "duplicate ATTACHMENTS_DIR"
        ATTACHMENTS_SEEN=true
        [[ -n "$value" ]] || fail "ATTACHMENTS_DIR must not be empty (remove the line to disable attachments)"
        [[ "$value" =~ ^/[A-Za-z0-9._/-]+$ && "$value" != */ && "/$value/" != */../* && "/$value/" != */./* && "$value" != *//* ]] || fail "ATTACHMENTS_DIR must be a plain absolute path without . or .. segments or a trailing slash"
        ATTACHMENTS_DIR="$value" ;;
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

# Runs "$@" with libpq PG* variables derived from $DB_DSN, so the DSN and its
# password never appear on a command line (visible in `ps`) or in logs.
with_pg_env() {
  DB_DSN="$DB_DSN" python3 -c '
import os
import sys
import urllib.parse

uri = urllib.parse.urlsplit(os.environ["DB_DSN"])
if uri.scheme not in {"postgres", "postgresql"} or not uri.hostname or not uri.username or not uri.path.strip("/"):
    raise SystemExit("unsupported PostgreSQL DSN")
query_pairs = urllib.parse.parse_qsl(uri.query, keep_blank_values=True)
query = dict(query_pairs)
if len(query_pairs) != len(query) or set(query) - {"sslmode"}:
    raise SystemExit("unsupported or duplicate PostgreSQL DSN query parameter")
env = os.environ.copy()
env.update({
    "PGHOST": uri.hostname,
    "PGPORT": str(uri.port or 5432),
    "PGDATABASE": urllib.parse.unquote(uri.path.lstrip("/")),
    "PGUSER": urllib.parse.unquote(uri.username),
    "PGPASSWORD": urllib.parse.unquote(uri.password or ""),
})
if "sslmode" in query:
    env["PGSSLMODE"] = query["sslmode"]
os.execvpe(sys.argv[1], sys.argv[1:], env)
' "$@"
}

# Database name of a postgres:// URL (no decoding; plain names only).
dsn_database_name() {
  local rest="${1#*://}"
  rest="${rest#*/}"
  printf '%s' "${rest%%\?*}"
}

# Integration tests reset their database, so they may only ever run against a
# dedicated database whose name ends in _test and differs from production.
load_test_database() {
  local uid owner mode_bits line key raw value test_db prod_db count=0
  [[ -f "$TEST_ENV_FILE" && ! -L "$TEST_ENV_FILE" ]] || fail "test database file missing: create $TEST_ENV_FILE (mode 600) containing TEST_DATABASE_URL so integration tests can run"
  uid="$(id -u)"; owner="$(stat -c %u "$TEST_ENV_FILE")"; mode_bits="$(stat -c %a "$TEST_ENV_FILE")"
  [[ "$owner" == "$uid" ]] || fail "test database file must be owned by deployment user uid $uid"
  (( (8#$mode_bits & 8#077) == 0 )) || fail "test database file must not grant group/other permissions (use chmod 600)"
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^(TEST_DATABASE_URL)=(.*)$ ]] || fail "test database file contains an unsupported or malformed assignment"
    key="${BASH_REMATCH[1]}"; raw="${BASH_REMATCH[2]}"
    if [[ "$raw" =~ ^\'([^\']*)\'$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^\"([^\"\\]*)\"$ ]]; then value="${BASH_REMATCH[1]}"
    elif [[ "$raw" =~ ^[^[:space:]\#\'\"\\]*$ ]]; then value="$raw"
    else fail "test database value is outside the supported literal syntax"
    fi
    count=$((count + 1))
  done < "$TEST_ENV_FILE"
  (( count == 1 )) || fail "test database file must define TEST_DATABASE_URL exactly once"
  [[ "$value" =~ ^postgres(ql)?://[^/?#[:space:]]+/[A-Za-z0-9_]+(\?[^[:space:]]*)?$ ]] || fail "TEST_DATABASE_URL must be a postgres:// URL with a plain database name"
  test_db="$(dsn_database_name "$value")"
  prod_db="$(dsn_database_name "$DB_DSN")"
  [[ "$test_db" == *_test ]] || fail "test database name must end in _test"
  [[ "$test_db" != "$prod_db" ]] || fail "test database must not be the production database"
  TEST_DB_DSN="$value"
}
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

# Succeeds when the space-separated systemd ReadWritePaths value ($1) contains
# $2 or one of its parent directories. Entries may carry a "-" (ignore missing)
# or "+" prefix.
unit_allows_write() {
  local entry dir="$2"
  local -a entries
  read -ra entries <<<"$1"
  for entry in "${entries[@]}"; do
    entry="${entry#[-+]}"
    entry="${entry%/}"
    [[ -n "$entry" ]] || continue
    [[ "$dir" == "$entry" || "$dir" == "$entry"/* ]] && return 0
  done
  return 1
}

# The attachment directory must be private to the deployment user, live outside
# the release tree, and be writable by the hardened service unit.
check_attachments_dir() {
  [[ -n "$ATTACHMENTS_DIR" ]] || { log "ATTACHMENTS_DIR not set; attachments stay disabled"; return 0; }
  local uid owner mode_bits rw
  [[ "$ATTACHMENTS_DIR/" != "$ROOT_DIR/"* ]] || fail "ATTACHMENTS_DIR must be outside $ROOT_DIR so releases and checkouts never touch it"
  [[ -d "$ATTACHMENTS_DIR" && ! -L "$ATTACHMENTS_DIR" ]] || fail "ATTACHMENTS_DIR must be an existing directory, not a symlink"
  uid="$(id -u)"; owner="$(stat -c %u "$ATTACHMENTS_DIR")"; mode_bits="$(stat -c %a "$ATTACHMENTS_DIR")"
  [[ "$owner" == "$uid" ]] || fail "ATTACHMENTS_DIR must be owned by deployment user uid $uid"
  (( (8#$mode_bits & 8#007) == 0 )) || fail "ATTACHMENTS_DIR must not grant other-user permissions (use chmod 750)"
  rw="$(sudo_systemctl show --property=ReadWritePaths --value)" || fail "cannot query service ReadWritePaths"
  unit_allows_write "$rw" "$ATTACHMENTS_DIR" || fail "$SERVICE_NAME cannot write ATTACHMENTS_DIR: install the unit with ReadWritePaths (docs/DEPLOYMENT.md, Lampiran)"
  log "attachments directory verified (writable by $SERVICE_NAME)"
}

# Snapshots every stored attachment (UUID-named files only; in-progress
# .upload-* temp files are skipped) into $1. Stored files are never modified
# after upload, so hard links are a complete copy at no extra disk cost; across
# filesystems the files are copied instead. The snapshot is checked file by
# file before it is moved into place.
snapshot_attachments() {
  local dest="$1" partial="$1.partial" src_list dst_list
  local uuid='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}'
  [[ ! -e "$dest" && ! -L "$dest" ]] || return 1
  rm -rf -- "$partial"
  mkdir -m 0700 -- "$partial" || return 1
  local -a files
  mapfile -d '' files < <(find "$ATTACHMENTS_DIR" -mindepth 1 -maxdepth 1 -type f -regextype posix-extended -regex ".*/$uuid" -print0)
  if ((${#files[@]} > 0)); then
    if [[ "$(stat -c %d "$ATTACHMENTS_DIR")" == "$(stat -c %d "$partial")" ]]; then
      cp -al -t "$partial" -- "${files[@]}" || { rm -rf -- "$partial"; return 1; }
    else
      cp -a -t "$partial" -- "${files[@]}" || { rm -rf -- "$partial"; return 1; }
    fi
  fi
  src_list="$(cd "$ATTACHMENTS_DIR" && find . -mindepth 1 -maxdepth 1 -type f -regextype posix-extended -regex "\./$uuid" -printf '%f %s\n' | sort)"
  dst_list="$(cd "$partial" && find . -mindepth 1 -maxdepth 1 -type f -printf '%f %s\n' | sort)"
  [[ "$src_list" == "$dst_list" ]] || { rm -rf -- "$partial"; return 1; }
  mv -T -- "$partial" "$dest" || { rm -rf -- "$partial"; return 1; }
}

# Keeps only the newest $BACKUPS_TO_KEEP attachment snapshots.
prune_attachment_snapshots() {
  local -a old
  local d
  mapfile -t old < <(find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -name 'pre-*.attachments' -printf '%T@ %p\n' | sort -rn | tail -n +"$((BACKUPS_TO_KEEP + 1))" | cut -d' ' -f2-)
  for d in "${old[@]}"; do
    [[ "$d" == "$BACKUP_DIR"/pre-*.attachments ]] && rm -rf -- "$d"
  done
  return 0
}

preflight() {
  log "=== PRE-FLIGHT ==="
  [[ "$mode" == check || "$ROOT_DIR" == "$REMOTE_DIR" ]] || fail "remote checkout mismatch: expected $REMOTE_DIR"
  local cmd dirty available_mb root_dev releases_parent_dev
  for cmd in git go gofmt node npm sqlc goose psql pg_dump pg_restore curl python3 sudo systemctl ss stat df mktemp tar sed sort grep readlink ln mv install timeout id cut cp chmod date find wc; do require_command "$cmd"; done
  parse_environment
  load_test_database
  dirty="$(git -C "$ROOT_DIR" status --porcelain)"; [[ -z "$dirty" ]] || fail "working tree is dirty; preserve or commit changes before deployment"
  SOURCE_HEAD="$(git -C "$ROOT_DIR" rev-parse --verify HEAD)"
  git -C "$ROOT_DIR" diff --quiet && git -C "$ROOT_DIR" diff --cached --quiet || fail "working tree is dirty"
  sudo_systemctl show --property=MainPID --value >/dev/null || fail "sudoers must permit systemctl show for $SERVICE_NAME"
  sudo -n ss -H -ltnp "sport = :$APP_PORT" >/dev/null || fail "sudoers must permit ss listener inspection"
  BUILD_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/informatika-deploy.XXXXXX")"
  with_pg_env timeout 5 psql -X -v ON_ERROR_STOP=1 -Atqc "SELECT 1" >/dev/null || fail "database connectivity check failed"
  if find "$ROOT_DIR/backend/sql/migrations" -maxdepth 1 -type f -name '*.sql' -print -quit | grep -q .; then
    # `goose status` would create the goose_db_version table, so a check-only run must use `validate`, which never connects.
    timeout 10 goose -dir "$ROOT_DIR/backend/sql/migrations" validate >/dev/null || fail "goose validate failed"
  else
    log "no Goose migration files yet; connectivity verified"
  fi
  available_mb="$(df -Pm "$ROOT_DIR" | sed -n '2{s/  */ /g;p}' | cut -d' ' -f4)"
  [[ "$available_mb" =~ ^[0-9]+$ ]] || fail "could not determine free disk space"
  (( available_mb >= MIN_DISK_MB )) || fail "free disk ${available_mb}MB is below ${MIN_DISK_MB}MB"
  root_dev="$(stat -c %d "$ROOT_DIR")"; releases_parent_dev="$root_dev"
  [[ ! -e "$RELEASES_DIR" ]] || releases_parent_dev="$(stat -c %d "$RELEASES_DIR")"
  [[ "$root_dev" == "$releases_parent_dev" ]] || fail "release and stable links must be on the same filesystem"
  check_port_owner
  check_attachments_dir
  log "pre-flight passed at $SOURCE_HEAD (secrets redacted)"
}

build_and_verify() {
  log "=== ISOLATED BUILD AND VERIFY ==="
  local src="$BUILD_ROOT/source"
  mkdir -p "$src" "$BUILD_ROOT/npm-cache" "$BUILD_ROOT/go-cache" "$BUILD_ROOT/go-mod-cache"
  git -C "$ROOT_DIR" archive --format=tar "$SOURCE_HEAD" | tar -xf - -C "$src"
  local unformatted
  unformatted="$(cd "$src/backend" && gofmt -l cmd internal)"
  [[ -z "$unformatted" ]] || fail "Go files are not gofmt-formatted: $(printf '%s' "$unformatted" | tr '\n' ' ')"
  (cd "$src/backend" && GOCACHE="$BUILD_ROOT/go-cache" GOMODCACHE="$BUILD_ROOT/go-mod-cache" sqlc generate && TEST_DATABASE_URL="$TEST_DB_DSN" REQUIRE_TEST_DB=1 go test ./... -count=1 && go vet ./... && go build -o "$BUILD_ROOT/api" ./cmd/api && go build -o "$BUILD_ROOT/adminctl" ./cmd/adminctl)
  [[ -x "$BUILD_ROOT/api" && -x "$BUILD_ROOT/adminctl" ]] || fail "backend binaries were not created"
  (cd "$src/frontend" && npm_config_cache="$BUILD_ROOT/npm-cache" npm ci && npm run lint && npm run typecheck && npm test && npm run build)
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
  local target="$1" link="$2" tmp
  tmp="$link.tmp.$$"
  ln -s -- "$target" "$tmp"
  mv -Tf -- "$tmp" "$link"
}

# Writes a verified, owner-only custom-format dump of the application database.
backup_database() {
  local dest="$1" partial="$1.partial"
  rm -f -- "$partial"
  (umask 077 && with_pg_env pg_dump --format=custom --no-owner --no-privileges --file="$partial") || { rm -f -- "$partial"; return 1; }
  [[ -s "$partial" ]] || { rm -f -- "$partial"; return 1; }
  pg_restore --list "$partial" >/dev/null || { rm -f -- "$partial"; return 1; }
  chmod 0600 -- "$partial"
  mv -f -- "$partial" "$dest"
}

# Keeps only the newest $BACKUPS_TO_KEEP pre-deploy dumps.
prune_backups() {
  local -a old
  mapfile -t old < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name 'pre-*.dump' -printf '%T@ %p\n' | sort -rn | tail -n +"$((BACKUPS_TO_KEEP + 1))" | cut -d' ' -f2-)
  ((${#old[@]} == 0)) || rm -f -- "${old[@]}"
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
  [[ -z "$BACKUP_FILE" ]] || log "pre-deploy database backup: $BACKUP_FILE (restore steps: docs/DEPLOYMENT.md)"
  [[ -z "$ATTACHMENTS_SNAPSHOT" ]] || log "pre-deploy attachment snapshot: $ATTACHMENTS_SNAPSHOT"
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

  # Back up before anything changes. Failure here aborts with nothing modified.
  if find "$ROOT_DIR/backend/sql/migrations" -maxdepth 1 -type f -name '*.sql' -print -quit | grep -q .; then
    mkdir -p -m 0700 "$BACKUP_DIR"
    BACKUP_FILE="$BACKUP_DIR/pre-$RELEASE_ID.dump"
    backup_database "$BACKUP_FILE" || { BACKUP_FILE=""; fail "database backup failed; nothing was changed"; }
    log "database backup written: $BACKUP_FILE ($(stat -c %s "$BACKUP_FILE") bytes)"
    prune_backups
  fi
  if [[ -n "$ATTACHMENTS_DIR" ]]; then
    mkdir -p -m 0700 "$BACKUP_DIR"
    ATTACHMENTS_SNAPSHOT="$BACKUP_DIR/pre-$RELEASE_ID.attachments"
    snapshot_attachments "$ATTACHMENTS_SNAPSHOT" || { ATTACHMENTS_SNAPSHOT=""; fail "attachment snapshot failed; nothing was changed"; }
    log "attachment snapshot written: $ATTACHMENTS_SNAPSHOT ($(find "$ATTACHMENTS_SNAPSHOT" -type f | wc -l) files)"
    prune_attachment_snapshots
  fi

  MUTATION_STARTED=true
  mkdir -p "$BIN_DIR" "$release_dir/bin" "$release_dir/public"
  install -m 0755 "$BUILD_ROOT/api" "$release_dir/bin/informatika-monitoring-api"
  install -m 0755 "$BUILD_ROOT/adminctl" "$release_dir/bin/adminctl"
  cp -a "$source_out/." "$release_dir/public/"
  chmod -R a-w "$release_dir"
  if find "$ROOT_DIR/backend/sql/migrations" -maxdepth 1 -type f -name '*.sql' -print -quit | grep -q .; then
    GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DB_DSN" goose -dir "$ROOT_DIR/backend/sql/migrations" up
  else
    log "no Goose migration files; skipping migration step"
  fi
  atomic_link "$release_dir/bin" "$BIN_DIR/current"
  atomic_link "$release_dir/public" "$PUBLIC_LINK"
  sudo_systemctl restart
  poll_ready
  BASE_URL="http://127.0.0.1:$APP_PORT" bash "$BUILD_ROOT/source/scripts/smoke.sh"
  MUTATION_STARTED=false
  trap - ERR
  log "deployed immutable release $RELEASE_ID; previous release targets retained"
  if find "$ROOT_DIR/backend/sql/migrations" -maxdepth 1 -type f -name '*.sql' -print -quit | grep -q .; then
    GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DB_DSN" goose -dir "$ROOT_DIR/backend/sql/migrations" version || true
  else
    log "Goose version: no migrations"
  fi
}

preflight
build_and_verify
if [[ "$mode" == check ]]; then
  if find "$ROOT_DIR/backend/sql/migrations" -maxdepth 1 -type f -name '*.sql' -print -quit | grep -q .; then
    backup_database "$BUILD_ROOT/rehearsal.dump" || fail "backup rehearsal failed"
    log "backup rehearsal passed ($(stat -c %s "$BUILD_ROOT/rehearsal.dump") bytes, discarded)"
  fi
  if [[ -n "$ATTACHMENTS_DIR" ]]; then
    snapshot_attachments "$BUILD_ROOT/attachments-rehearsal" || fail "attachment snapshot rehearsal failed"
    log "attachment snapshot rehearsal passed ($(find "$BUILD_ROOT/attachments-rehearsal" -type f | wc -l) files, discarded)"
  fi
  log "CHECK ONLY: used temporary source/build/cache; no migration, release write, symlink swap, restart, or runtime mutation performed"
  exit 0
fi
deploy_release
