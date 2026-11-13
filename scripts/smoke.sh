#!/usr/bin/env bash
set -Eeuo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8300}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf -- "$TMP_DIR"' EXIT

fetch() { curl --fail --silent --show-error --max-time 5 "$@"; }

fetch "$BASE_URL/healthz" >"$TMP_DIR/health.json"
fetch "$BASE_URL/readyz" >"$TMP_DIR/ready.json"
fetch --dump-header "$TMP_DIR/index.headers" "$BASE_URL/" >"$TMP_DIR/index.html"
fetch "$BASE_URL/progres/" >"$TMP_DIR/progres.html"
fetch "$BASE_URL/admin/login/" >"$TMP_DIR/login.html"

# The admin API must be mounted (it only is when the database is configured) and
# must refuse anonymous callers. These two calls are expected to return 401/405,
# so they are fetched without --fail and their status is asserted below.
status_of() { curl --silent --show-error --max-time 5 --output "$2" --write-out '%{http_code}' "${@:3}" "$1"; }
printf '%s' "$(status_of "$BASE_URL/api/v1/admin/auth/me" "$TMP_DIR/me.json")" >"$TMP_DIR/me.status"
printf '%s' "$(status_of "$BASE_URL/api/v1/admin/auth/login" "$TMP_DIR/login-get.json")" >"$TMP_DIR/login-get.status"

python3 - "$TMP_DIR" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
read = lambda name: (root / name).read_text(encoding="utf-8")

health = json.loads(read("health.json"))
ready = json.loads(read("ready.json"))
index = read("index.html")
progres = read("progres.html")
login = read("login.html")

assert health == {"status": "ok"}, health
assert ready == {"status": "ok", "database": "ok"}, ready
assert 'aria-label="Menu utama"' in index, "frontend marker not found"

# If a page policy is sent (CSP_MODE enforce or report-only), it must allow
# only same-origin and hashed inline scripts.
headers = read("index.headers").lower().splitlines()
policies = [h.split(":", 1)[1].strip() for h in headers if h.startswith(("content-security-policy:", "content-security-policy-report-only:"))]
for policy in policies:
    script_src = next((p.strip() for p in policy.split(";") if p.strip().startswith("script-src")), "")
    assert script_src.startswith("script-src 'self'"), policy
    assert "'unsafe-inline'" not in script_src and "'unsafe-eval'" not in script_src, script_src
    assert "frame-ancestors 'none'" in policy and "object-src 'none'" in policy, policy
assert "Progres Tugas Kelompok" in progres, "progress page marker not found"
assert 'name="username"' in login and 'name="password"' in login, "admin login form not found"

assert read("me.status") == "401", "anonymous /me must be 401, got " + read("me.status")
assert json.loads(read("me.json"))["error"]["code"] == "UNAUTHENTICATED", read("me.json")
assert read("login-get.status") == "405", "GET on login must be 405, got " + read("login-get.status")
assert json.loads(read("login-get.json"))["error"]["code"] == "METHOD_NOT_ALLOWED", read("login-get.json")
PY

printf '[smoke] health, readiness, public pages, admin login page, and anonymous admin API: PASS\n'
