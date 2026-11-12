# Informatika Monitoring — Progress

> Dokumen status aktual implementasi. Update setelah setiap task, verifikasi, commit, provisioning, dan deployment.
>
> Last updated: 12 November 2026

## Ringkasan

- Fase aktif: **Fase 0 — Foundation**
- Progress task: **6/12 selesai (50%)**
- Task terakhir selesai: **0.6 — Go static export serving**
- Next task: **0.7 — Goose/sqlc/environment skeleton**
- Local branch: `main`
- Local HEAD: `95609ab`
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
| 0.4 | Next.js static frontend | DONE | task commit | npm ci, lint, typecheck, static build, out/index.html, production audit PASS; independent review PASS |
| 0.5 | Typed frontend API client | DONE | task commit | lint, typecheck, static build, production audit, and real Go-through-Next rewrite smoke PASS; independent review PASS |
| 0.6 | Go static export serving | DONE | task commit | strict RED; targeted/full tests, vet, build, real-binary static smoke PASS; independent security review blockers fixed |
| 0.7 | Goose/sqlc/environment skeleton | PENDING | — | Belum dimulai |
| 0.8 | OpenAPI dan Makefile | PENDING | — | Belum dimulai |
| 0.9 | Hardened LMS-style deploy script | PENDING | — | Belum dimulai |
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
- Unknown routes return safe JSON `404`.

Not available yet:

- Authentication.
- Business database schema.
- Classes/groups CRUD.
- Real monitoring data.
- Group portal.
- Static frontend serving.
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
3. Public hostname/HTTPS cutover is not yet decided and is not needed for Tasks 0.6–0.9.

## Current Next Step

Task 0.7 — add Goose/sqlc and environment skeleton:

1. Add empty migration/query directories and deterministic sqlc configuration.
2. Add backend/frontend/root environment examples without real credentials.
3. Add optional development PostgreSQL Compose configuration without provisioning the VPS database.
4. Run `sqlc compile` and existing verification gates on `cbt-vps`.
5. Run independent review before commit.

## Update Rule

After each task:

1. Update the tracker row and summary percentage.
2. Record RED evidence for behavior implemented with TDD.
3. Record exact remote verification commands/results.
4. Record independent review blockers and final verdict.
5. Record commit hash.
6. Advance `Current Next Step`.
7. Never mark live DB/deploy/manual acceptance as PASS unless directly verified.
