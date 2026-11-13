#!/usr/bin/env bash
# Unit tests for the security-sensitive helper functions in deploy/deploy.sh.
# They are extracted by name and run in subshells, so deploy.sh itself is never
# executed and nothing outside a temporary directory is touched.
set -u

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
DEPLOY="$HERE/../deploy.sh"
WORK="$(mktemp -d)"
trap 'rm -rf -- "$WORK"' EXIT

pass=0
failed=0

extract() {
  awk -v name="$1" '$0 ~ "^" name "\\(\\) \\{" { p = 1 } p { print } p && /^}$/ { exit }' "$DEPLOY"
}

FUNCS=""
for fn in dsn_database_name load_test_database parse_environment with_pg_env; do
  body="$(extract "$fn")"
  [[ -n "$body" ]] || { echo "FATAL: function $fn not found in deploy.sh" >&2; exit 2; }
  FUNCS+="$body"$'\n'
done

PROD_DSN='postgres://app:s3cret@127.0.0.1:5432/informatika_monitoring?sslmode=disable'

# check NAME EXPECT(ok|fail) COMMAND... runs COMMAND in a subshell with the
# extracted functions and a fail() that exits 1.
check() {
  local name="$1" expect="$2"; shift 2
  local out rc
  out="$(
    (
      set -Eeuo pipefail
      log() { :; }
      fail() { echo "fail: $*" >&2; exit 1; }
      eval "$FUNCS"
      "$@"
    ) 2>&1
  )"
  rc=$?
  if { [[ "$expect" == ok && $rc -eq 0 ]] || [[ "$expect" == fail && $rc -ne 0 ]]; }; then
    pass=$((pass + 1))
  else
    failed=$((failed + 1))
    printf 'NOT OK: %s (expected %s, exit %s)\n%s\n' "$name" "$expect" "$rc" "$out" >&2
  fi
}

# ---- dsn_database_name -------------------------------------------------------
t_dbname() { [[ "$(dsn_database_name "$1")" == "$2" ]] || fail "got $(dsn_database_name "$1")"; }
check "dbname with query" ok t_dbname 'postgres://u:p@h:5432/mydb?sslmode=disable' mydb
check "dbname without query" ok t_dbname 'postgres://u:p@h/mydb' mydb
check "dbname postgresql scheme" ok t_dbname 'postgresql://u@h/mydb_test?x=1' mydb_test

# ---- load_test_database ------------------------------------------------------
write_test_env() { # content mode
  local f="$WORK/test.env"
  rm -f -- "$f"
  printf '%b' "$1" >"$f"
  chmod "$2" "$f"
  printf '%s' "$f"
}

run_load() { # file -> prints TEST_DB_DSN
  TEST_ENV_FILE="$1"
  DB_DSN="${2:-$PROD_DSN}"
  TEST_DB_DSN=""
  load_test_database
  printf '%s' "$TEST_DB_DSN"
}
t_load_expect() { [[ "$(run_load "$1" "${3:-}")" == "$2" ]] || fail "unexpected DSN"; }

GOOD='postgres://t:pw@127.0.0.1:5432/informatika_monitoring_test?sslmode=disable'
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\n" 600)"
check "load: unquoted value" ok t_load_expect "$f" "$GOOD"
f="$(write_test_env "TEST_DATABASE_URL='$GOOD'\n" 600)"
check "load: single quoted" ok t_load_expect "$f" "$GOOD"
f="$(write_test_env "TEST_DATABASE_URL=\"$GOOD\"\n" 600)"
check "load: double quoted" ok t_load_expect "$f" "$GOOD"
f="$(write_test_env "# comment\n\nTEST_DATABASE_URL=$GOOD\n" 600)"
check "load: comments and blanks" ok t_load_expect "$f" "$GOOD"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\r\n" 600)"
check "load: CRLF" ok t_load_expect "$f" "$GOOD"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\n" 400)"
check "load: mode 400" ok t_load_expect "$f" "$GOOD"

f="$(write_test_env "TEST_DATABASE_URL=$GOOD\n" 644)"
check "load: mode 644 rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\n" 640)"
check "load: mode 640 rejected" fail run_load "$f"
check "load: missing file rejected" fail run_load "$WORK/does-not-exist"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\n" 600)"
ln -sf "$f" "$WORK/link.env"
check "load: symlink rejected" fail run_load "$WORK/link.env"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\nTEST_DATABASE_URL=$GOOD\n" 600)"
check "load: duplicate rejected" fail run_load "$f"
f="$(write_test_env "# nothing here\n" 600)"
check "load: missing assignment rejected" fail run_load "$f"
f="$(write_test_env "DATABASE_URL=$GOOD\n" 600)"
check "load: wrong key rejected" fail run_load "$f"
f="$(write_test_env "export TEST_DATABASE_URL=$GOOD\n" 600)"
check "load: export prefix rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=$GOOD\nOTHER=1\n" 600)"
check "load: extra key rejected" fail run_load "$f"

f="$(write_test_env "TEST_DATABASE_URL=postgres://t:pw@127.0.0.1:5432/informatika_monitoring?sslmode=disable\n" 600)"
check "load: production database rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=postgres://t:pw@127.0.0.1:5432/informatika_dev?sslmode=disable\n" 600)"
check "load: name without _test rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=postgres://t:pw@127.0.0.1:5432/prod_test?sslmode=disable\n" 600)"
check "load: same as production _test rejected" fail run_load "$f" 'postgres://a:b@127.0.0.1:5432/prod_test'
f="$(write_test_env "TEST_DATABASE_URL=postgres://t:pw@127.0.0.1:5432/foo_test;drop\n" 600)"
check "load: injection characters rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=postgres://t:pw@127.0.0.1:5432/a/b_test\n" 600)"
check "load: extra path segment rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=mysql://t:pw@127.0.0.1:3306/x_test\n" 600)"
check "load: non-postgres scheme rejected" fail run_load "$f"
f="$(write_test_env "TEST_DATABASE_URL=\n" 600)"
check "load: empty value rejected" fail run_load "$f"

# ---- parse_environment -------------------------------------------------------
run_parse() { # content mode
  ENV_FILE="$WORK/app.env"
  rm -f -- "$ENV_FILE"
  printf '%b' "$1" >"$ENV_FILE"
  chmod "$2" "$ENV_FILE"
  APP_PORT=8300
  PUBLIC_LINK="/srv/app/public"
  APP_ADDR="" PUBLIC_DIR="" DB_DSN="" ADMIN_COOKIE_SECURE="" CSP_MODE=""
  parse_environment
  printf '%s' "$ADMIN_COOKIE_SECURE"
}
run_parse_csp() { run_parse "$1" 600 >/dev/null; printf '%s' "$CSP_MODE"; }
t_csp_expect() { [[ "$(run_parse_csp "$1")" == "$2" ]] || fail "unexpected CSP_MODE"; }
BASE_ENV="APP_ADDR=:8300\nPUBLIC_DIR=/srv/app/public\nDATABASE_URL=$PROD_DSN\n"
t_parse_expect() { [[ "$(run_parse "$1" 600)" == "$2" ]] || fail "unexpected ADMIN_COOKIE_SECURE"; }
check "env: without ADMIN_COOKIE_SECURE" ok t_parse_expect "$BASE_ENV" ""
check "env: ADMIN_COOKIE_SECURE=true" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=true\n" true
check "env: ADMIN_COOKIE_SECURE=false" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=false\n" false
check "env: ADMIN_COOKIE_SECURE=1" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=1\n" 1
check "env: ADMIN_COOKIE_SECURE=0" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=0\n" 0
check "env: ADMIN_COOKIE_SECURE=TRUE normalised" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=TRUE\n" true
check "env: ADMIN_COOKIE_SECURE quoted" ok t_parse_expect "${BASE_ENV}ADMIN_COOKIE_SECURE=\"true\"\n" true
check "env: ADMIN_COOKIE_SECURE=maybe rejected" fail run_parse "${BASE_ENV}ADMIN_COOKIE_SECURE=maybe\n" 600
check "env: ADMIN_COOKIE_SECURE=yes rejected" fail run_parse "${BASE_ENV}ADMIN_COOKIE_SECURE=yes\n" 600
check "env: ADMIN_COOKIE_SECURE empty rejected" fail run_parse "${BASE_ENV}ADMIN_COOKIE_SECURE=\n" 600
check "env: duplicate ADMIN_COOKIE_SECURE rejected" fail run_parse "${BASE_ENV}ADMIN_COOKIE_SECURE=true\nADMIN_COOKIE_SECURE=false\n" 600
check "env: unknown key rejected" fail run_parse "${BASE_ENV}SECRET_KEY=abc\n" 600
check "env: shell expansion rejected" fail run_parse "${BASE_ENV}ADMIN_COOKIE_SECURE=\$(id)\n" 600
check "env: group-readable rejected" fail run_parse "$BASE_ENV" 640
check "env: missing DATABASE_URL rejected" fail run_parse "APP_ADDR=:8300\nPUBLIC_DIR=/srv/app/public\n" 600
check "env: both DB variables rejected" fail run_parse "${BASE_ENV}GOOSE_DBSTRING=$PROD_DSN\n" 600
check "env: CSP_MODE absent" ok t_csp_expect "$BASE_ENV" ""
check "env: CSP_MODE=enforce" ok t_csp_expect "${BASE_ENV}CSP_MODE=enforce\n" enforce
check "env: CSP_MODE=report-only" ok t_csp_expect "${BASE_ENV}CSP_MODE=report-only\n" report-only
check "env: CSP_MODE=off" ok t_csp_expect "${BASE_ENV}CSP_MODE=off\n" off
check "env: CSP_MODE=Report-Only normalised" ok t_csp_expect "${BASE_ENV}CSP_MODE=Report-Only\n" report-only
check "env: CSP_MODE=disabled rejected" fail run_parse "${BASE_ENV}CSP_MODE=disabled\n" 600
check "env: CSP_MODE empty rejected" fail run_parse "${BASE_ENV}CSP_MODE=\n" 600
check "env: duplicate CSP_MODE rejected" fail run_parse "${BASE_ENV}CSP_MODE=off\nCSP_MODE=enforce\n" 600
check "env: wrong port rejected" fail run_parse "APP_ADDR=:9000\nPUBLIC_DIR=/srv/app/public\nDATABASE_URL=$PROD_DSN\n" 600

# ---- with_pg_env -------------------------------------------------------------
t_pg_env_exposes_vars() {
  local dump
  dump="$(DB_DSN='postgres://us%40er:p%2Fw@db.example:6543/my_db?sslmode=require' with_pg_env env)"
  grep -qx 'PGHOST=db.example' <<<"$dump" || fail "PGHOST"
  grep -qx 'PGPORT=6543' <<<"$dump" || fail "PGPORT"
  grep -qx 'PGDATABASE=my_db' <<<"$dump" || fail "PGDATABASE"
  grep -qx 'PGUSER=us@er' <<<"$dump" || fail "PGUSER"
  grep -qx 'PGPASSWORD=p/w' <<<"$dump" || fail "PGPASSWORD"
  grep -qx 'PGSSLMODE=require' <<<"$dump" || fail "PGSSLMODE"
}
t_pg_env_default_port() {
  DB_DSN='postgres://u:p@h/d' with_pg_env env | grep -qx 'PGPORT=5432' || fail "default port"
}
t_pg_env_not_on_argv() { # the password must never be a command-line argument
  local args
  args="$(DB_DSN='postgres://u:topsecret@h/d' with_pg_env printf '%s\n' a b)"
  ! grep -q topsecret <<<"$args" || fail "password reached argv"
}
t_pg_env_exit_code() { ! DB_DSN='postgres://u:p@h/d' with_pg_env false; }
t_pg_env_bad_dsn() { ! DB_DSN="$1" with_pg_env true; }
check "pg env: variables derived and decoded" ok t_pg_env_exposes_vars
check "pg env: default port" ok t_pg_env_default_port
check "pg env: password not on argv" ok t_pg_env_not_on_argv
check "pg env: exit status propagates" ok t_pg_env_exit_code
check "pg env: rejects mysql scheme" ok t_pg_env_bad_dsn 'mysql://u:p@h/d'
check "pg env: rejects missing user" ok t_pg_env_bad_dsn 'postgres://h/d'
check "pg env: rejects missing database" ok t_pg_env_bad_dsn 'postgres://u:p@h/'
check "pg env: rejects unknown query parameter" ok t_pg_env_bad_dsn 'postgres://u:p@h/d?options=-c%20x'
check "pg env: rejects duplicate query parameter" ok t_pg_env_bad_dsn 'postgres://u:p@h/d?sslmode=disable&sslmode=require'

printf '%d passed, %d failed\n' "$pass" "$failed"
[[ "$failed" -eq 0 ]]
