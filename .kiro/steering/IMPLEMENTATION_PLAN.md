# Fase 0 — Foundation Implementation Plan

> **Untuk Hermes:** gunakan TDD dan checkpoint task-by-task. Jangan lanjut ke Fase 1 sebelum seluruh completion gate Fase 0 hijau dan user menyetujui checkpoint.

**Status:** PROPOSED — menunggu approval implementasi
**Goal:** Membuat fondasi monorepo Go + Next.js yang bisa dibangun, diuji, dan dideploy secara aman ke `cbt-vps`, tanpa mengimplementasikan autentikasi atau CRUD bisnis lebih awal.
**Architecture:** Next.js App Router menjadi UI-only dan menghasilkan static export. Go `chi` menjadi satu-satunya backend bisnis serta production runtime: melayani `/api/v1/*`, health endpoints, dan static export frontend. PostgreSQL tersedia sebagai dependency workspace, tetapi Fase 0 belum membuat schema bisnis.
**Tech stack:** Go, `net/http`, `go-chi/chi/v5`, Next.js App Router, TypeScript, Tailwind CSS, PostgreSQL, `pgx/v5`, sqlc, Goose, systemd, Playwright smoke.
**Design:** `DESIGN.md` + antislop mode DURING.
**Environment agreement:** coding + Git lokal; install/build/test/migrate/runtime di `cbt-vps`.

---

## 1. Scope

### Included

- Git + monorepo baseline.
- Frontend Next.js static-export baseline.
- Backend Go HTTP baseline.
- `/healthz`, `/readyz`, dan `GET /api/v1/public/summary` placeholder contract.
- Go static-file serving + extensionless route fallback.
- Development rewrite `/api/*` dari Next.js ke Go.
- PostgreSQL development/workspace configuration.
- Goose/sqlc directory and config skeleton.
- OpenAPI baseline.
- Makefile and environment examples.
- LMS-style hardened deploy script adapted for `cbt-vps`.
- systemd unit template and deployment documentation.
- One designed public landing/monitoring shell using dummy labels only, not fake student records.
- Automated checks and remote smoke.

### Explicitly excluded

- Admin login/session.
- Group access codes.
- Business tables/migrations.
- CRUD kelas/kelompok.
- Real monitoring records.
- Gmail integration or upload.
- Public domain cutover.
- Real student data.

## 2. Status Legend

- `[ ]` Pending
- `[~]` In progress
- `[x]` Done and verified
- `[!]` Blocked, with evidence

## 3. Locked Foundation Decisions

1. Root project remains a monorepo with `frontend/`, `backend/`, `deploy/`, and `docs/`.
2. Backend/business API is Go only.
3. Next.js uses `output: "export"`; no production Node server.
4. Go binary serves static frontend and API from one origin.
5. Internal port is `8300`.
6. VPS target is `cbt-vps`; checkout `/home/ubuntu/informatika-monitoring`.
7. systemd service is `informatika-monitoring-api.service`.
8. Database is `informatika_monitoring`, with a least-privilege role created during provisioning.
9. No source deploy/restart occurs without user approval at the checkpoint.
10. No generated visual assets in Fase 0; typography, CSS, and semantic icons only.

## 4. Task-by-Task Plan

### Task 0.1 — Initialize repository hygiene `[x]`

**Objective:** Create a clean Git baseline and directory conventions without generated/runtime files.

**Files:**
- Create: `.gitignore`
- Create: `README.md`
- Create: `LOCAL_AI_CONTEXT.md`
- Create directories: `frontend/`, `backend/`, `deploy/`, `docs/`

**Steps:**
1. Initialize Git on branch `main`.
2. Add narrow ignores for `.env`, Node/Go build output, `frontend/out`, runtime `public`, coverage, IDE files, and local temporary files.
3. Write `LOCAL_AI_CONTEXT.md` with paths, local/VPS agreement, deploy target, phase pointer, and commands placeholders.
4. Verify only intended planning/docs files are tracked.
5. Commit: `chore: initialize monitoring project repository`.

**Verify:**
```bash
git status --short
git check-ignore frontend/node_modules frontend/out backend/bin .env
```

**Acceptance:** planning docs remain present; no secret or generated output is tracked.

### Task 0.2 — Scaffold Go module and router with TDD `[x]`

**Objective:** Establish the Go service and prove routing before implementation grows.

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/api/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/http/router.go`
- Create: `backend/internal/http/router_test.go`

**RED tests:**
- `GET /healthz` returns `200` and safe JSON.
- `GET /api/v1/public/summary` returns the exact placeholder shape.
- Unknown `/api/v1/*` returns JSON `404`, not the frontend shell.
- Request ID header exists.

**Run RED:**
```bash
cd backend && go test ./internal/http -count=1
```
Expected: FAIL because router package/handlers do not exist.

**GREEN implementation:**
- `chi.NewRouter()` with recovery, request ID, safe timeout, and content type.
- `/healthz` is process liveness only.
- Summary placeholder contains zero counts plus `updatedAt: null` and explicit `manualUpdate: true`; no fake groups.
- Config parses `APP_ADDR`, `DATABASE_URL`, `PUBLIC_DIR`, and environment without logging secrets.

**Run GREEN:**
```bash
cd backend && go test ./internal/http -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/api
```

**Commit:** `feat(api): add Go HTTP foundation and health endpoint`.

### Task 0.3 — Add PostgreSQL readiness boundary `[x]`

**Objective:** Keep liveness independent while readiness accurately reports DB state.

**Files:**
- Create: `backend/internal/db/pool.go`
- Create: `backend/internal/health/service.go`
- Create: `backend/internal/health/service_test.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/cmd/api/main.go`

**RED tests:**
- Readiness returns `503` when DB pinger fails.
- Readiness returns `200` when DB pinger succeeds.
- Response exposes only component status, never DSN or raw driver errors.
- `/healthz` remains `200` when DB is unavailable.

**Run RED:**
```bash
cd backend && go test ./internal/health ./internal/http -count=1
```

**GREEN implementation:**
- Define minimal `Pinger` interface.
- Initialize bounded `pgxpool` in `main`.
- `/readyz` performs bounded ping and emits safe JSON.
- Configure HTTP server read-header/read/write/idle timeouts and graceful shutdown.

**Verify:**
```bash
cd backend && go test ./... -count=1 && go vet ./... && go build ./cmd/api
```

**Commit:** `feat(api): add PostgreSQL readiness check`.

### Task 0.4 — Scaffold Next.js static frontend `[ ]`

**Objective:** Establish a production-exportable UI-only frontend with no direct database or server action.

**Files:**
- Create via scaffold: `frontend/package.json`, lockfile, TypeScript/Next/Tailwind configs
- Modify: `frontend/next.config.ts`
- Create: `frontend/src/app/layout.tsx`
- Create: `frontend/src/app/page.tsx`
- Create: `frontend/src/app/globals.css`
- Create: `frontend/src/lib/api-client/public.ts`
- Create: `frontend/src/types/public.ts`

**Requirements:**
- `output: "export"`.
- No Route Handlers, Server Actions, Prisma, Supabase client, or DB environment variable.
- Dev rewrite `/api/:path*` to configurable Go origin.
- Fonts loaded with controlled fallback; build must not require secret values.
- Metadata and `lang="id"` are correct.

**UI direction:**
- Build a compact homepage introduction and monitoring preview shell.
- Show the manual-update warning prominently.
- Represent the three separate status lanes.
- Do not create fake student rows, fake KPI statistics, testimonials, decorative hero art, or non-functional buttons.
- Navigation only links to routes that exist in Fase 0.

**Verify on VPS:**
```bash
cd frontend
npm ci
npm run lint
npm run typecheck
npm run build
test -f out/index.html
```

**Commit:** `feat(web): add static public foundation`.

### Task 0.5 — Wire typed frontend API client `[ ]`

**Objective:** Prove frontend-to-Go contract without embedding business logic in Next.js.

**Files:**
- Modify: `frontend/src/lib/api-client/public.ts`
- Create: `frontend/src/components/public/ProjectSummary.tsx`
- Create: `frontend/src/components/public/ProjectSummary.test.tsx` if Vitest is selected
- Modify: `frontend/src/app/page.tsx`

**Requirements:**
- Client calls relative `/api/v1/public/summary`.
- Runtime has explicit loading, empty/zero, error, and success states.
- Error copy is actionable and does not expose internal response details.
- The zero-data state is honest and explains that groups have not been configured.

**Verify:**
```bash
cd frontend && npm run typecheck && npm run lint && npm run build
```

**Manual smoke:** start Go + dev frontend on VPS; page request reaches Go through rewrite and renders the zero state.

**Commit:** `feat(web): connect public summary to Go API`.

### Task 0.6 — Serve static export from Go `[ ]`

**Objective:** Make the production Go binary own both API and frontend routes.

**Files:**
- Create: `backend/internal/web/static.go`
- Create: `backend/internal/web/static_test.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/cmd/api/main.go`

**RED tests:**
- `/` serves `index.html`.
- Extensionless `/monitoring` resolves `monitoring.html` when present.
- Missing frontend route returns the intended 404 document, not API JSON.
- `/api/v1/missing` remains JSON 404 and never falls back to HTML.
- Path traversal is rejected.

**Run RED:**
```bash
cd backend && go test ./internal/web ./internal/http -count=1
```

**GREEN implementation:**
- Register API routes before static fallback.
- Resolve exact asset, `<route>.html`, `<route>/index.html`, then `404.html`.
- Set cache policy: hashed static assets cacheable; HTML no-cache/controlled.
- `PUBLIC_DIR` points to deployed static directory.

**Verify:**
```bash
cd backend && go test ./... -count=1 && go vet ./... && go build ./cmd/api
```

**Commit:** `feat(api): serve exported frontend routes`.

### Task 0.7 — Add Goose/sqlc and environment skeleton `[ ]`

**Objective:** Lock database tooling without prematurely creating business schema.

**Files:**
- Create: `backend/sql/migrations/.gitkeep`
- Create: `backend/sql/queries/.gitkeep`
- Create: `backend/sqlc.yaml`
- Create: `backend/.env.example`
- Create: `frontend/.env.example`
- Create: `.env.example`
- Create: `docker-compose.yml`

**Requirements:**
- Local compose exposes PostgreSQL for optional developer use, but remote VPS remains required verification environment.
- No real password or host credential in examples.
- sqlc output lives under ignored `backend/internal/dbsql/` or another explicit generated path.
- Goose migration path is deterministic.
- `DATABASE_URL` exists only in backend/root runtime environment, never frontend.

**Verify:**
```bash
# on cbt-vps with pinned tools
cd backend && sqlc compile
```

**Commit:** `chore(db): add PostgreSQL tooling baseline`.

### Task 0.8 — Add OpenAPI and Makefile contract `[ ]`

**Objective:** Give frontend/backend one testable contract and consistent commands.

**Files:**
- Create: `docs/openapi.yaml`
- Create: `Makefile`
- Modify: `README.md`

**OpenAPI scope:**
- `/healthz`
- `/readyz`
- `/api/v1/public/summary`
- Standard error envelope

**Make targets:**
- `test-backend`
- `vet-backend`
- `build-backend`
- `lint-frontend`
- `typecheck-frontend`
- `build-frontend`
- `verify`
- `deploy` (dispatch only; no implicit live deploy from verify)

**Verify:**
- Validate OpenAPI syntax with a pinned/living tool available on VPS.
- Run `make verify` and require nonzero exit on any failed gate.

**Commit:** `docs: add API and verification contract`.

### Task 0.9 — Create hardened LMS-style deploy script `[ ]`

**Objective:** Adapt the proven LMS deployment pattern without carrying unrelated R2 assumptions.

**Files:**
- Create: `deploy/deploy.sh`
- Create: `deploy/informatika-monitoring-api.service`
- Create: `docs/DEPLOYMENT.md`
- Create: `scripts/smoke.sh`

**Local dispatch defaults:**
```text
REMOTE_HOST=cbt-vps
REMOTE_DIR=/home/ubuntu/informatika-monitoring
BRANCH=main
```

**Remote pre-flight:**
- `.env` exists and required vars are non-empty.
- PostgreSQL reachable without printing DSN.
- Go/Node/npm/Goose/sqlc versions available.
- Disk free >= 500 MB.
- Port 8300 is free or owned by the current service.
- Working tree/HEAD behavior is explicit; script must not silently reset dirty changes.

**Build gates before mutation:**
- `go test ./... -count=1`
- `go vet ./...`
- Go binary built under a temp directory.
- `npm ci`, lint, typecheck, static build.
- Goose status/version check.

**Deploy:**
- Timestamped backup of current binary and static directory/manifest as appropriate.
- Copy frontend `out/.` to a generated `public` directory without quoted-glob bugs.
- Run Goose migrations.
- Copy new binary as `.new`, then atomic rename.
- Restart `informatika-monitoring-api`.

**Verify + rollback:**
- Poll `http://127.0.0.1:8300/readyz` 10 times, 2-second interval.
- Smoke `/healthz`, `/readyz`, `/`, and a known frontend marker.
- On failure restore prior binary/static state and restart.
- Report deployed Git HEAD and migration status.

**Security:** script never prints `.env`, DSN, password, cookie keys, or full command expansion containing secrets.

**Commit:** `chore(deploy): add hardened single-VPS deployment`.

### Task 0.10 — Provision `cbt-vps` workspace `[ ]`

**Objective:** Prepare an isolated non-public workspace using the approved paths.

**Prerequisite:** explicit user approval to mutate `cbt-vps`.

**Actions:**
1. Check OS, disk, ports, PostgreSQL, Go, Node, npm, Goose, sqlc, and systemd state.
2. Create bare repo `/home/ubuntu/git-repos/informatika-monitoring.git`.
3. Create checkout `/home/ubuntu/informatika-monitoring` tracking the bare repo.
4. Create PostgreSQL role and database with generated secret stored only in `.env`.
5. Install/copy systemd unit; keep service disabled until binary and env are valid.
6. Configure environment with restrictive file permissions.
7. Enable/start only after build and migration gates pass.
8. Do not attach a public domain yet.

**Verify:**
- DB role cannot create superusers/databases.
- `.env` mode is restrictive.
- Service runs as `ubuntu`, restarts automatically, and cannot expose secrets in status.
- Port 8300 listens only as intended.

**Commit:** none for server secrets; commit only generic templates/docs.

### Task 0.11 — Remote baseline deployment and smoke `[ ]`

**Objective:** Prove the entire delivery chain on `cbt-vps`.

**Prerequisite:** Tasks 0.1–0.10 complete and explicit deploy approval.

**Flow:**
1. Commit local intended files.
2. Push GitHub `origin main` after repository destination is confirmed/created.
3. Push bare remote `server main`.
4. Fast-forward VPS checkout; preserve any remote dirt rather than deleting it silently.
5. Run `bash deploy/deploy.sh --remote`.
6. Run `scripts/smoke.sh` against `127.0.0.1:8300`.
7. Capture evidence: local commit, server HEAD, systemd status, Goose version, healthz, readyz, root marker.

**Acceptance:** deployment is reproducible and no manual binary-copy workaround is required.

**Commit:** `docs: record Fase 0 deployment evidence` if an evidence document is added.

### Task 0.12 — Antislop Delivery Gate and phase close `[ ]`

**Objective:** Close Fase 0 only after functionality, design, accessibility, and deployment truth are verified.

**Checks:**

#### Block A — Hard gate
- No Gmail integration/upload/AI grading/direct frontend DB.
- No fabricated users, stats, testimonials, or trust claims.
- Every visible link/button works.
- Semantic HTML, keyboard, focus, `lang=id`, heading hierarchy.
- Contrast computation passes; reduced motion handled.
- 360px, intermediate, desktop, and 200% zoom checked.

#### Block B — Purpose gate
- Every color, type, status indicator, surface, and layout treatment serves a documented purpose in `DESIGN.md`.
- No generic SaaS template clusters.

#### Block C — Quality locks
- Loading, zero/empty, error, and success states exist.
- Mobile is a real reflow, not squeezed desktop.
- Status uses text + icon + color.

#### Block D — Engineering delivery
- Go test/vet/build PASS.
- Frontend lint/typecheck/build PASS.
- Static routes and API boundary tests PASS.
- Remote deploy + smoke PASS.
- PLAN, DESIGN, OpenAPI, README, DEPLOYMENT, and LOCAL_AI_CONTEXT are current.

**Output:** create `docs/F0_ACCEPTANCE.md` with PASS/FAIL evidence and exact unresolved items.

**Commit:** `docs: close Fase 0 foundation` only if all required gates pass.

## 5. Fase 0 Completion Gate

- [ ] Repository initialized and clean.
- [ ] Go service test/vet/build green.
- [ ] Next.js lint/typecheck/static build green.
- [ ] API and static frontend served from one Go binary.
- [ ] `/healthz` and `/readyz` semantics verified.
- [ ] PostgreSQL workspace isolated.
- [ ] Goose/sqlc/OpenAPI baseline valid.
- [ ] Hardened deploy script proves rollback-ready path.
- [ ] `cbt-vps` deployed HEAD, systemd, migration, health, readiness, and frontend marker verified.
- [ ] Antislop Delivery Gate PASS.
- [ ] No Fase 1 business/auth work leaked into foundation.
- [ ] User approves moving to Fase 1.

## 6. Risks and Mitigations

1. **Static export route behavior** — test extensionless `.html` resolution in Go before deploy.
2. **Remote toolchain drift** — inspect and pin compatible stable versions during Task 0.10; record in `LOCAL_AI_CONTEXT.md`.
3. **Deploy script mutates before validation** — all test/build gates precede migration/swap/restart.
4. **Rollback restores binary but not frontend** — backup/restore static state or deploy release directories with a current symlink.
5. **Empty database mistaken for failure** — summary zero state is valid; readiness depends on DB connection, not record count.
6. **Public data leakage later** — establish safe DTO and API boundary in Fase 0; never reuse admin DTO for public routes.
7. **Design becomes generic after scaffold** — `DESIGN.md` and delivery gate are mandatory inputs/checks, not optional polish.

## 7. Implementation Notes

- 2026-11-12: plan created from `PLAN.md`, LMS deployment pattern, `DESIGN.md`, and upstream antislop rules.
- 2026-11-12: user approved progress. Task 0.1 completed: Git `main` initialized; `.gitignore`, `README.md`, and `LOCAL_AI_CONTEXT.md` created; ignore rules verified for secrets and generated output; `DESIGN.md` lint remains green.
- 2026-11-12: user approved a build-only VPS workspace. Go 1.27.4, Node 26.13.1 LTS, and npm 11.20.0 installed on `cbt-vps`; source workspace created without DB, `.env`, systemd, port binding, or runtime activation.
- 2026-11-12: Task 0.2 strict RED observed remotely: `internal/http` failed with `no non-test Go files`. Minimal router/config/server implementation then passed targeted test, full `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd/api`. Independent review found no blockers.
- 2026-11-12: Task 0.3 strict RED observed remotely: `internal/health` had no production file and `NewRouterWithPinger` was undefined. Fake-Pinger readiness tests then passed targeted/full Go gates. Independent review found and drove fixes for premature DB activation, missing nil-dependency `/readyz`, and shutdown cleanup races; final re-review PASS. Real PostgreSQL smoke remains explicitly pending Task 0.10.
- `project-2` had no Git repository before Task 0.1.
- `cbt-vps` was previously cleaned; database, systemd, and runtime provisioning still require explicit side-effect approval at Task 0.10.

## 8. Current Next Step

**CHECKPOINT AFTER TASK 0.3.**

1. Commit the verified readiness boundary.
2. Start Task 0.4: scaffold Next.js App Router + TypeScript + Tailwind with static export in `frontend/`.
3. Implement only the honest public foundation from `DESIGN.md`; no fake student rows, KPI claims, generated visual assets, auth, or business CRUD.
4. Run `npm ci`, lint, typecheck, and static build on `cbt-vps`; verify `out/index.html` exists.
5. Real DB, `.env`, bare deploy repo, systemd, port binding, and runtime activation remain out of scope.
