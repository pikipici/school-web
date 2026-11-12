# Deployment — Informatika Monitoring

## Boundary

Target: SSH alias `cbt-vps`, checkout `/home/ubuntu/informatika-monitoring`, user `ubuntu`, unit `informatika-monitoring-api.service`, port `8300`. Task 0.9 supplies tooling only; it does not activate the service.

## Server prerequisites

Install Git, Go, sqlc, Node/npm, Goose, PostgreSQL client, curl, Python 3, systemd, `ss`, and standard GNU utilities. The checkout must be clean. `deploy/deploy.sh` refuses every tracked or untracked change.

Create `/home/ubuntu/informatika-monitoring/.env`, owned by `ubuntu`, mode `0600`. The deploy parser never sources this file. It accepts only literal systemd-compatible assignments for these keys (blank lines/comments are allowed; values may be unquoted or simply single/double quoted, with no escapes or expansion):

```text
APP_ADDR=:8300
PUBLIC_DIR=/home/ubuntu/informatika-monitoring/public
DATABASE_URL='postgres://...'
```

`GOOSE_DBSTRING` may replace `DATABASE_URL`, but do not define both. DSNs are passed through environment/config files, never command arguments, and are not logged. The service reads the same EnvironmentFile. `PUBLIC_DIR` is the stable `public` symlink.

Install `deploy/informatika-monitoring-api.service`; its ExecStart follows the stable `bin/current` symlink. Before first activation, a deployment must create a release and these links.

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

`--check` validates inputs, files, permissions, dependencies, disk/filesystem, DB connectivity, Goose status, service/port ownership, and all test/vet/sqlc/build/npm gates. It exports the captured Git HEAD into a temporary directory and puts Go/npm caches there, so the checkout is not dirtied. It may download build dependencies into that temporary area. It does not write releases, migrate, switch links, restart the unit, or mutate runtime state.

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

Before mutation, deployment captures HEAD, enforces a clean tree, checks every command (including sqlc), verifies DB connectivity and `goose status`, validates the configured address/port and listener ownership against the unit's MainPID/cgroup, and passes all isolated build gates. It then rechecks HEAD and cleanliness.

A deployment creates immutable `releases/<timestamp>-<commit>/{bin,public}`, runs forward Goose migrations, atomically switches `bin/current` and `public`, restarts, polls readiness, and runs smoke tests. Previous release directories and prior link targets remain available. Release and links must share a filesystem.

## Failure and rollback

Once release mutation starts, an ERR trap attempts rollback. Rollback atomically restores both prior symlinks, restarts the service, and verifies readiness. It reports success only if all those steps succeed; otherwise it explicitly reports an unverified/unhealthy rollback and exits nonzero. A first deployment has no previous runtime to restore.

Database migrations are forward-only in automatic rollback. Migrations must be backward-compatible; database restore is a separate operator runbook.

## Manual verification

```bash
sudo systemctl status informatika-monitoring-api.service --no-pager -l
sudo journalctl -u informatika-monitoring-api.service -n 100 --no-pager
curl -fsS http://127.0.0.1:8300/healthz
curl -fsS http://127.0.0.1:8300/readyz
BASE_URL=http://127.0.0.1:8300 bash scripts/smoke.sh
```
