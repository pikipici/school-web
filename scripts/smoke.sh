#!/usr/bin/env bash
set -Eeuo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8300}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf -- "$TMP_DIR"' EXIT

curl --fail --silent --show-error --max-time 5 "$BASE_URL/healthz" >"$TMP_DIR/health.json"
curl --fail --silent --show-error --max-time 5 "$BASE_URL/readyz" >"$TMP_DIR/ready.json"
curl --fail --silent --show-error --max-time 5 "$BASE_URL/" >"$TMP_DIR/index.html"

python3 - "$TMP_DIR" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
health = json.loads((root / "health.json").read_text(encoding="utf-8"))
ready = json.loads((root / "ready.json").read_text(encoding="utf-8"))
index = (root / "index.html").read_text(encoding="utf-8")

assert health == {"status": "ok"}, health
assert ready == {"status": "ok", "database": "ok"}, ready
assert "Papan Progres Informatika" in index, "frontend marker not found"
PY

printf '[smoke] health, readiness, and frontend marker: PASS\n'
