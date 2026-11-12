# Deployment — Informatika Monitoring

## Boundary

Target: SSH alias `cbt-vps`, checkout `/home/ubuntu/informatika-monitoring`, user `ubuntu`, unit `informatika-monitoring-api.service`, port `8300`. Task 0.9 supplied the tooling; Tasks 0.10–0.11 provisioned and activated the service (internal port only, no public hostname/HTTPS).

## Server prerequisites

Install Git, Go, sqlc, Node/npm, Goose, PostgreSQL client (including `pg_dump` and `pg_restore`), curl, Python 3, systemd, `ss`, and standard GNU utilities. The checkout must be clean. `deploy/deploy.sh` refuses every tracked or untracked change.

Create `/home/ubuntu/informatika-monitoring/.env`, owned by `ubuntu`, mode `0600`. The deploy parser never sources this file. It accepts only literal systemd-compatible assignments for these keys (blank lines/comments are allowed; values may be unquoted or simply single/double quoted, with no escapes or expansion):

```text
APP_ADDR=:8300
PUBLIC_DIR=/home/ubuntu/informatika-monitoring/public
DATABASE_URL='postgres://...'
ADMIN_COOKIE_SECURE=false
```

`ADMIN_COOKIE_SECURE` is optional and must be `true`, `false`, `1`, or `0` (anything else aborts the deploy). Keep it `false` while the site is served over plain HTTP. Set it to `true` only once HTTPS is in front of the app: the admin session cookie then becomes `__Host-admin_session` with the `Secure` flag, and over plain HTTP it would not be sent at all.

`GOOSE_DBSTRING` may replace `DATABASE_URL`, but do not define both. DSNs are passed through environment/config files, never command arguments, and are not logged. The service reads the same EnvironmentFile. `PUBLIC_DIR` is the stable `public` symlink.

Install `deploy/informatika-monitoring-api.service`; its ExecStart follows the stable `bin/current` symlink. Before first activation, a deployment must create a release and these links.

## Integration test database

Before building, `deploy.sh` runs the Go integration tests against a dedicated database, with `REQUIRE_TEST_DB=1` so they can never be skipped silently. Those tests drop and recreate the `public` schema of that database, so the script only accepts a database whose name ends in `_test` and differs from the production database.

Create `/home/ubuntu/.informatika-monitoring-test.env` (owner `ubuntu`, mode `0600`, outside the repository) with exactly one line:

```text
TEST_DATABASE_URL=postgres://informatika_monitoring_test:<password>@127.0.0.1:5432/informatika_monitoring_test?sslmode=disable
```

The role must not be a superuser and must have no access to the production database. Override the location with `TEST_ENV_FILE`. The deploy aborts, before changing anything, if the file is missing, group/other-readable, a symlink, malformed, or points at a non-`_test` or production database.

## Minimal noninteractive sudo

Use `visudo -f /etc/sudoers.d/informatika-deploy` and adapt command paths from `command -v systemctl` and `command -v ss` if needed:

```sudoers
Cmnd_Alias INFORMATIKA_SYSTEMCTL = /usr/bin/systemctl show --property=MainPID --value informatika-monitoring-api.service, /usr/bin/systemctl show --property=ControlGroup --value informatika-monitoring-api.service, /usr/bin/systemctl restart informatika-monitoring-api.service
Cmnd_Alias INFORMATIKA_SS = /usr/bin/ss -H -ltnp sport = :8300
ubuntu ALL=(root) NOPASSWD: INFORMATIKA_SYSTEMCTL, INFORMATIKA_SS
```

Do not grant unrestricted `systemctl`, a shell, or passwordless sudo generally. The script always uses `sudo -n`. Some sudo versions tokenize the `ss` filter differently; verify the exact command with `sudo -n ss -H -ltnp 'sport = :8300'` and narrowly adjust that one sudoers entry if required.

## Check-only validation

From the server checkout:

```bash
bash -n deploy/deploy.sh
bash -n scripts/smoke.sh
bash deploy/deploy.sh --check
```

`--check` validates inputs, files, permissions, dependencies, disk/filesystem, DB connectivity, migration files (`goose validate`, which never connects to a database), service/port ownership, and all gates: Go test (including integration tests), vet, sqlc, both Go binaries, and frontend lint, typecheck, unit tests, and build. When migration files exist it also rehearses the pre-deploy backup into a temporary file (a read-only `pg_dump`) and discards it. It exports the captured Git HEAD into a temporary directory and puts Go/npm caches there, so the checkout is not dirtied. It may download build dependencies into that temporary area. It does not write releases, migrate, switch links, restart the unit, or mutate runtime state.

## Deployment

Local safe dispatch validates the SSH host and shell-quotes the remote path:

```bash
bash deploy/deploy.sh
# or: make deploy
```

Directly on the server:

```bash
cd /home/ubuntu/informatika-monitoring
bash deploy/deploy.sh --remote
```

Before mutation, deployment captures HEAD, enforces a clean tree, checks every command (including sqlc), verifies DB connectivity and the migration files, validates the configured address/port and listener ownership against the unit's MainPID/cgroup, and passes all isolated build gates. It then rechecks HEAD and cleanliness.

Whenever migration files exist, deployment first writes a verified, owner-only (`0600`) custom-format dump to `backups/pre-<release>.dump` and keeps the newest `BACKUPS_TO_KEEP` (default 10). If the backup fails, the deploy aborts with nothing changed. `backups/` and `*.dump` are git-ignored and must stay out of version control.

A deployment creates immutable `releases/<timestamp>-<commit>/{bin,public}` (the API binary and `adminctl`), runs forward Goose migrations, atomically switches `bin/current` and `public`, restarts, polls readiness, and runs smoke tests. Previous release directories and prior link targets remain available. Release and links must share a filesystem.

## HTTPS (Caddy)

Caddy terminates TLS on ports 80 and 443 and proxies to the app on `127.0.0.1:8300`. It is installed from the official Caddy apt repository, and `/etc/caddy/Caddyfile` is a copy of `deploy/Caddyfile` (`sudo install -m 0644 -o root -g root deploy/Caddyfile /etc/caddy/Caddyfile`, then `caddy validate --config /etc/caddy/Caddyfile`). The Caddyfile overwrites `X-Forwarded-For` with the real client address, drops `X-Real-IP`, removes the `Server` header, adds security headers, and caps request bodies at 1 MB. The app trusts `X-Forwarded-For` only from a loopback peer and reads its rightmost entry, so this overwrite is what makes the login rate limit see the true client.

Turn-on order. Do not reorder: starting Caddy before DNS and the firewall are right makes it fail its certificate request repeatedly, and Let's Encrypt limits failed attempts.

1. DNS (Cloudflare): an `A` record for `fikriramaaulia.online` pointing to the server's public IP, proxy status **DNS only (grey cloud)**. Remove the existing proxied `A`/`AAAA` records for that name first; a proxied record hides the server and breaks certificate issuance. This replaces whatever the name serves today.
2. Firewall of the VPS provider's console (not the OS firewall, which is inactive): allow inbound TCP 80 and 443 from anywhere. Port 8300 must stay closed to the internet.
3. Check from outside the server: `dig +short fikriramaaulia.online` returns the server IP, and TCP 80 and 443 connect.
4. `sudo systemctl enable --now caddy`, then `sudo journalctl -u caddy -n 50 --no-pager` and look for a successful certificate. `curl -I https://fikriramaaulia.online/` must return 200 with a valid certificate, and `curl -I http://fikriramaaulia.online/` must redirect to HTTPS.
5. Only after step 4 works, set `ADMIN_COOKIE_SECURE=true` in `.env` and restart the service. From then on the admin cookie is `__Host-admin_session` with `Secure`, and the app accepts admin requests only from an `https://` origin, so the old SSH-tunnel login over plain `http://localhost` stops working. Use the HTTPS address.
6. After a few stable days raise HSTS in the Caddyfile (`max-age=86400`, later `31536000`). Test with `caddy validate`, install, `sudo systemctl reload caddy`.

Rollback: `sudo systemctl disable --now caddy`, set `ADMIN_COOKIE_SECURE=false`, restart the app, and use the SSH tunnel again. Browsers that already saw the HSTS header will refuse plain HTTP for its `max-age`, which is why it starts at one hour.

A Content-Security-Policy header is not set yet: the static Next.js export uses inline scripts, so a policy needs per-build hashes or a nonce setup. Track it as follow-up work.

## Failure and rollback

Once release mutation starts, an ERR trap attempts rollback. Rollback atomically restores both prior symlinks, restarts the service, and verifies readiness. It reports success only if all those steps succeed; otherwise it explicitly reports an unverified/unhealthy rollback and exits nonzero. A first deployment has no previous runtime to restore.

Database migrations are forward-only in automatic rollback. Migrations must be backward-compatible. The rollback message names the pre-deploy backup so an operator can restore it if a migration itself was the problem.

## Restoring a database backup (manual, destructive)

Restoring replaces data in the application database with the state at backup time; anything written since is lost. Stop the service first, restore, then start it. Let libpq read the password from `PGPASSWORD` rather than the command line:

```bash
sudo systemctl stop informatika-monitoring-api.service
PGPASSWORD='...' pg_restore --clean --if-exists --no-owner \
  --host 127.0.0.1 --username informatika_monitoring_app --dbname informatika_monitoring \
  /home/ubuntu/informatika-monitoring/backups/pre-<release>.dump
sudo systemctl start informatika-monitoring-api.service
```

Prefer restoring into a scratch database first (`createdb`, restore, inspect) when time allows.

## Admin account

After the first deploy that ships the migrations, create the single admin on the server. `adminctl` asks for the password on the terminal (twice, hidden) and never accepts it as an argument or environment variable:

```bash
cd /home/ubuntu/informatika-monitoring
set -a; . ./.env; set +a
bin/current/adminctl create-admin --username fikri
# lost password (also ends every existing session):
bin/current/adminctl reset-password --username fikri
```

Piped input (`--password-stdin`) is supported for automation. Account changes are written to the `audit_log` table.

## Deploy-script tests

`make test-deploy` (or `bash deploy/tests/test_deploy_functions.sh`) unit-tests the security-sensitive helpers in `deploy.sh`: `.env` and test-database file parsing, the `_test` safeguard, and the libpq environment helper. It is part of `make verify` and touches nothing outside a temporary directory.

## Manual verification

```bash
sudo systemctl status informatika-monitoring-api.service --no-pager -l
sudo journalctl -u informatika-monitoring-api.service -n 100 --no-pager
curl -fsS http://127.0.0.1:8300/healthz
curl -fsS http://127.0.0.1:8300/readyz
BASE_URL=http://127.0.0.1:8300 bash scripts/smoke.sh
```

The smoke script checks health and readiness, the public pages, the admin login page, and that the admin API is mounted and refuses anonymous callers (`/me` returns 401, GET on `/login` returns 405).
