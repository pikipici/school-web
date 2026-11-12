# Informatika Monitoring — Progress

> Dokumen status aktual implementasi. Update setelah setiap task, verifikasi, commit, provisioning, dan deployment.
>
> Last updated: 12 November 2026

## Ringkasan

- Fase aktif: **Fase 0 — Foundation**
- Progress task: **9/12 selesai (75%)**
- Task terakhir selesai: **0.9 — Hardened LMS-style deploy script**
- Next task: **0.10 — Provision `cbt-vps` runtime workspace (menunggu approval eksplisit)**
- Local branch: `main`
- Task 0.8 implementation commit: `d332215`
- Working tree: bersih saat status ini dibuat
- Runtime aplikasi: **belum diaktifkan**
- Real database: **belum dibuat**
- Public domain/live deployment: **belum dilakukan**

## Sources of Truth

1. `PLAN.md` — spesifikasi produk, arsitektur, security, API, dan AC01–AC25.
2. `DESIGN.md` — design tokens, UX, responsive, accessibility, dan antislop rules.
3. `.kiro/steering/IMPLEMENTATION_PLAN.md` — task-by-task Fase 0.
4. `LOCAL_AI_CONTEXT.md` — environment, working agreements, dan pointer sesi.
5. `.kiro/steering/PROGRESS.md` — status aktual dan evidence pelaksanaan.

## Fase 0 Tracker

| Task | Scope | Status | Commit | Verification |
|---|---|---|---|---|
| 0.1 | Repository hygiene | DONE | `31d11c8` | Git `main`, ignore rules, DESIGN lint, diff check |
| 0.2 | Go router foundation | DONE | `258bf9a` | RED tercatat; targeted/full tests, vet, build PASS; independent review PASS |
| 0.3 | PostgreSQL readiness boundary | DONE | `95609ab` | RED tercatat; fake-Pinger tests, full tests, vet, build PASS; final independent review PASS |
| 0.4 | Next.js static frontend | DONE | `fe684f1` | npm ci, lint, typecheck, static build, out/index.html, production audit PASS; independent review PASS |
| 0.5 | Typed frontend API client | DONE | `b0c4aee` | lint, typecheck, static build, production audit, and real Go-through-Next rewrite smoke PASS; independent review PASS |
| 0.6 | Go static export serving | DONE | `700762a` | strict RED; targeted/full tests, vet, build, real-binary static smoke PASS; independent security review blockers fixed |
| 0.7 | Goose/sqlc/environment skeleton | DONE | `45f03b4` | sqlc v1.29 compile/generate, Go tests/vet/build PASS; environment boundary review PASS; Compose static review only (Docker unavailable) |
| 0.8 | OpenAPI dan Makefile | DONE | `d332215` | OpenAPI YAML parse; remote `make verify` PASS; independent review PASS |
| 0.9 | Hardened LMS-style deploy script | DONE | pending commit | Shell syntax, isolated remote `--check`, all build gates, non-mutation assertions, and independent re-review PASS |
| 0.10 | Provision `cbt-vps` runtime workspace | PENDING APPROVAL | — | DB, `.env`, bare repo, systemd, dan port belum dibuat |
| 0.11 | Remote baseline deployment dan smoke | PENDING APPROVAL | — | Belum deploy |
| 0.12 | Antislop Delivery Gate dan phase close | PENDING | — | Menunggu seluruh gate sebelumnya |

## Completed Evidence

### Task 0.1 — Repository hygiene

Selesai:

- Git repository diinisialisasi pada branch `main`.
- Membuat `.gitignore`, `README.md`, dan `LOCAL_AI_CONTEXT.md`.
- Mengunci `PLAN.md`, `DESIGN.md`, dan Fase 0 implementation plan.
- Ignore rules diverifikasi untuk `.env`, Node/Next output, Go binary/generated code, coverage, runtime `public`, dan temporary files.
- `@google/design.md lint DESIGN.md` PASS.
- `git diff --check` PASS.

Commit:

```text
31d11c8 chore: initialize informatika monitoring project
```

### Task 0.2 — Go HTTP foundation

Strict TDD RED:

```text
internal/http: no non-test Go files
```

Implemented:

- Go module.
- `chi` router.
- `GET /healthz`.
- Honest zero-state `GET /api/v1/public/summary`.
- Safe JSON 404 untuk unknown API route.
- Request ID middleware.
- Panic recovery.
- HTTP timeout baseline.
- Config parser tanpa secret logging.

Remote verification:

```text
go test ./internal/http -count=1  PASS
go test ./... -count=1           PASS
go vet ./...                     PASS
go build ./cmd/api               PASS
independent review               PASS, no blocker
```

Commit:

```text
258bf9a feat(api): add Go HTTP foundation and health endpoint
```

### Task 0.3 — PostgreSQL readiness boundary

Strict TDD RED:

```text
internal/health: no non-test Go files
undefined: httpapi.NewRouterWithPinger
```

Implemented:

- `health.Pinger` interface.
- Readiness service dengan safe status.
- `/readyz` mengembalikan `200` saat pinger sehat.
- `/readyz` mengembalikan safe `503` saat DB gagal atau dependency belum dikonfigurasi.
- `/healthz` tetap independen dari DB.
- Bounded readiness context 2 detik.
- `pgxpool` boundary dengan static/redacted config errors.
- Database hanya di-wire bila `DATABASE_URL` tersedia.
- Graceful shutdown + forced-close fallback.
- Server goroutine ditunggu sebelum resource cleanup.
- Header size limit dan HTTP timeout hardening.

Review loop:

1. Review pertama menemukan premature DB activation, missing nil-dependency `/readyz`, dan cleanup/shutdown issues.
2. Review kedua menemukan forced-close path belum menunggu server goroutine.
3. Seluruh blocker diperbaiki.
4. Final independent re-review: PASS.

Remote verification:

```text
go test ./internal/health ./internal/http -count=1  PASS
go test ./... -count=1                         PASS
go vet ./...                                   PASS
go build ./cmd/api                             PASS
final independent review                       PASS, no blocker
```

Commit:

```text
95609ab feat(api): add PostgreSQL readiness boundary
```

### Task 0.4 — Next.js static frontend

Implemented:

- Next.js App Router + TypeScript + Tailwind dengan `output: "export"`.
- Fondasi UI publik sesuai `DESIGN.md`, termasuk peringatan pembaruan manual dan tiga jalur status terpisah.
- Responsive/accessibility baseline, font lokal, metadata Bahasa Indonesia, serta honest empty state tanpa data/KPI palsu.

Remote verification:

```text
npm ci                  PASS
npm run lint            PASS
npm run typecheck       PASS
npm run build           PASS
out/index.html          PRESENT
npm audit --omit=dev    0 vulnerability
independent review      PASS, no blocker
```

Commit:

```text
fe684f1 feat(web): add static public foundation
```

### Task 0.5 — Typed frontend API client

Implemented:

- Typed relative client untuk `GET /api/v1/public/summary`.
- Defensive response validation.
- Accessible loading, error + retry, honest zero, dan success states.
- Development rewrite ke Go tanpa memindahkan business logic ke Next.js.

Remote verification:

```text
npm ci / lint / typecheck / build        PASS
production dependency audit              PASS, 0 vulnerability
real Go-through-Next rewrite smoke        PASS
independent review                        PASS, no blocker
```

Commit:

```text
b0c4aee feat(web): connect public summary to Go API
```

### Task 0.6 — Go static export serving

Strict TDD RED:

```text
internal/web: no non-test Go files
undefined: httpapi.NewRouterWithPublicDir
```

Implemented:

- Root, extensionless `.html`, directory-index fallback, dan custom frontend 404.
- API namespace tetap JSON, termasuk encoded/double-encoded path.
- Traversal dan stable symlink escape protection.
- GET/HEAD/range/metadata semantics melalui `http.ServeContent`.
- HTML `no-store`, hashed `/_next/static` immutable, aset lain `no-cache`.
- Portable MIME mapping dan `PUBLIC_DIR` runtime wiring.

Review loop:

1. Review menemukan symlink escape, open-after-status, API boundary, cache, dan MIME gaps.
2. Seluruh blocker direproduksi dengan RED tests lalu diperbaiki.
3. Regression test memastikan encoded API 404 menghasilkan JSON valid.

Remote verification:

```text
go test ./internal/web ./internal/http ./internal/config -count=1  PASS
go test ./... -count=1                                        PASS
go vet ./...                                                  PASS
go build ./cmd/api                                            PASS
real binary + actual Next out static smoke                    PASS
```

Commit:

```text
700762a feat(api): serve exported frontend routes
```

### Task 0.7 — Goose/sqlc/environment skeleton

Implemented:

- Deterministic `sqlc.yaml`, migration/query directories, dan infrastructure-only connectivity query.
- Root/backend/frontend environment examples tanpa real credentials.
- `DATABASE_URL` tetap backend-only.
- Optional local PostgreSQL Compose dengan loopback binding dan app role non-superuser.
- Generated sqlc output diarahkan ke ignored `backend/internal/dbsql/`.

Remote verification:

```text
sqlc v1.29 compile/generate  PASS
go test ./...                PASS
go vet ./...                 PASS
go build ./...               PASS
independent review           PASS, no blocker
docker compose runtime       NOT RUN — Docker unavailable on cbt-vps
```

Commit:

```text
45f03b4 chore(db): add PostgreSQL tooling baseline
```

### Task 0.8 — OpenAPI and Makefile contract

Implemented:

- OpenAPI 3.0.3 contract for `/healthz`, `/readyz`, `/api/v1/public/summary`, request IDs, and the standard JSON error envelope.
- Root Make targets for separate backend/frontend gates plus fail-fast `make verify`.
- Explicit `make deploy` dispatcher kept outside verification and guarded until Task 0.9 supplies the script.
- README API and verification guidance.
- Backend build output is written to ignored `backend/bin/api`.

Remote verification on an isolated source copy:

```text
OpenAPI YAML parse          PASS
make test-backend           PASS
make vet-backend            PASS
make build-backend          PASS
make lint-frontend          PASS
make typecheck-frontend     PASS
make build-frontend         PASS
independent review          PASS, no blocker
```

Notes:

- `npm ci` was required before the first frontend gate because dependencies were absent in the isolated copy.
- npm reported seven moderate dependency advisories; this is recorded as a follow-up signal, not silently treated as a clean production audit.
- Runtime, database, systemd, and port 8300 were not activated or mutated.

### Task 0.9 — Hardened deployment tooling

Implemented:

- Safe local SSH dispatcher and remote `--check` / `--remote` modes.
- Restricted `.env` parser that never executes shell content and rejects unsafe ownership/permissions.
- Secret-safe PostgreSQL/Goose invocation, sqlc/Goose pre-flight gates, clean-tree and pinned-HEAD enforcement.
- Isolated temporary build source/caches so check-only validation does not dirty source or mutate runtime.
- Immutable release directories with atomic `bin/current` and `public` symlink switches.
- systemd template, bounded readiness polling, frontend/API smoke script, and verified rollback path design.
- Deployment runbook including least-privilege sudo boundaries and forward-only migration caveat.

Non-activating remote verification:

```text
bash -n deploy/deploy.sh       PASS
bash -n scripts/smoke.sh       PASS
DB/Goose preflight harness     PASS (mocked; no real DB provisioned)
sqlc generate                 PASS
Go test/vet/build              PASS
npm ci/lint/typecheck/build    PASS
--check runtime non-mutation   PASS
independent initial review     blockers found and fixed
independent final re-review    PASS, no blocker
```

Notes:

- The first npm attempt hit a transient `ECONNRESET`; the unchanged rerun passed.
- npm still reports seven moderate dependency advisories.
- No database, `.env`, systemd installation, release directory, port listener, restart, migration, or runtime activation was performed.
- Actual migration/restart/rollback exercise remains pending Tasks 0.10–0.11 approval.

## VPS State

Target:

- SSH alias: `cbt-vps`
- Host: `43.134.111.212`
- User: `ubuntu`
- Build-only workspace: `/home/ubuntu/informatika-monitoring`

Verified toolchain:

```text
Go:      1.27.4
Node.js: 26.13.1 LTS
npm:     11.20.0
```

Current provisioning boundary:

- Build-only source workspace: PRESENT
- Application database `informatika_monitoring`: ABSENT
- Application `.env`: ABSENT
- Bare Git deploy repository: ABSENT
- `informatika-monitoring-api.service`: ABSENT
- Runtime port 8300 activation: NOT DONE
- Public/live deployment: NOT DONE

The absence above is intentional. Database, secret environment, systemd, port binding, and runtime activation require the Task 0.10 checkpoint.

## Current Backend Contract

Available in source:

- `GET /healthz`
  - Process liveness only.
  - Does not depend on PostgreSQL.
- `GET /readyz`
  - `200` when configured DB pinger succeeds.
  - Safe `503` when DB is missing/unavailable.
- `GET /api/v1/public/summary`
  - Honest zero state.
  - `totalGroups: 0`
  - `manualUpdate: true`
  - `updatedAt: null`
- Unknown API routes return safe JSON `404`.
- Go serves the exported Next.js frontend with extensionless routing, frontend 404, safe cache policy, and traversal protection.

Not available yet:

- Authentication.
- Business database schema.
- Classes/groups CRUD.
- Real monitoring data.
- Group portal.
- Deployed runtime.

## Safety and Scope Status

Still enforced:

- No Gmail API, IMAP, or automatic inbox sync.
- No PPT/PDF upload.
- No AI grading.
- No real student data.
- No direct frontend database access.
- No Next.js business Route Handlers or Server Actions.
- No runtime restart/deployment without approval.

## Blockers / Pending External Decisions

1. Real PostgreSQL smoke waits for Task 0.10 provisioning approval.
2. Runtime activation waits for Task 0.10.
3. Docker is unavailable on `cbt-vps`; optional local Compose has static review only and is not required for Tasks 0.8–0.9.
4. Public hostname/HTTPS cutover is not yet decided and is not needed for Tasks 0.8–0.9.

## Current Next Step

Task 0.10 — provisioning checkpoint, blocked pending explicit approval:

1. Re-inspect OS, disk, ports, PostgreSQL, toolchain, systemd, and current remote workspace state.
2. Create the bare deploy repository and proper Git checkout without discarding unexpected remote state.
3. Provision the least-privilege PostgreSQL role/database and restrictive `.env`.
4. Install the reviewed systemd unit and narrow sudo boundary.
5. Activate runtime only after all pre-flight/build/migration gates pass; public domain remains out of scope.

## Update Rule

After each task:

1. Update the tracker row and summary percentage.
2. Record RED evidence for behavior implemented with TDD.
3. Record exact remote verification commands/results.
4. Record independent review blockers and final verdict.
5. Record commit hash.
6. Advance `Current Next Step`.
7. Never mark live DB/deploy/manual acceptance as PASS unless directly verified.
