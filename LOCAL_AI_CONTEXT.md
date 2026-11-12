# Informatika Monitoring — Local AI Context

> Pointer operasional. Update setelah setiap task/phase atau perubahan environment.

## TL;DR

- Produk: monitoring dan evaluasi manual tugas kelompok Informatika kelas IX.
- Backend wajib Go (`chi`, `pgxpool`, `sqlc`, Goose, PostgreSQL).
- Frontend wajib Next.js App Router + TypeScript + Tailwind, static export.
- Tidak ada Gmail API/IMAP, upload PPT/PDF, atau AI grading.
- Local Windows = coding + Git; build/test/migrate/runtime = `cbt-vps`.

## Paths

- Local: `D:\Program\project-2`
- VPS SSH alias: `cbt-vps` (`ubuntu@43.134.111.212`)
- VPS checkout target: `/home/ubuntu/informatika-monitoring`
- VPS bare repo target: `/home/ubuntu/git-repos/informatika-monitoring.git`
- Service target: `informatika-monitoring-api.service`
- Internal port: `8300`
- Database target: `informatika_monitoring`
- GitHub remote: `https://github.com/pikipici/school-web.git` (`origin`, public)

## Sources of truth

1. `PLAN.md` — product, data, API, security, acceptance AC01–AC25.
2. `DESIGN.md` — visual direction and antislop rules.
3. `.kiro/steering/IMPLEMENTATION_PLAN.md` — current execution checklist.
4. `.kiro/steering/PROGRESS.md` — status aktual, evidence, commit, blocker, dan next step.
5. This file — environment and current checkpoint.

## Working agreements

- Read `PLAN.md`, `DESIGN.md`, and current task before coding.
- TDD for production behavior: test must fail for the expected reason before implementation.
- Do not run local builds/tests; use `cbt-vps`.
- Do not mutate/restart deployed runtime without explicit approval.
- Preserve remote dirty state; never silently `reset --hard` over changes.
- Commit each verified task separately.
- Use dummy data only; never real student records during development.

## Current state

- Fase 0 plan approved for progress on 2026-11-12.
- Tasks 0.1–0.11 complete: repository baseline, Go HTTP/readiness, Next.js static public foundation/client, secure Go static serving, Goose/sqlc/environment skeleton, OpenAPI/Makefile contract, hardened deployment tooling, VPS provisioning, and verified baseline deployment (release `20261112T140834Z-e779d364ec24`).
- Backend exposes `/healthz`, safe `/readyz`, and honest zero-state `/api/v1/public/summary`; unknown API routes return JSON 404.
- Readiness uses a fake-Pinger-tested boundary and is verified against the real PostgreSQL on `cbt-vps` (`/readyz` returns database ok).
- GitHub `origin` configured and `main` pushed to `pikipici/school-web`.
- `cbt-vps` has a bare repo, clean Git checkout, least-privilege PostgreSQL role/database, `.env` mode 0600, matching hardened systemd unit, enabled service, immutable release, and verified port 8300 runtime at HEAD `e779d36`.
- Toolchain installed for remote verification: Go 1.27.4, Node 26.13.1 LTS, npm 11.20.0.
- VPS was previously cleaned of the old Crypto Station project.

## Phase tracker

- [~] Fase 0 — Foundation
  - [x] 0.1 Repository hygiene
  - [x] 0.2 Go router foundation
  - [x] 0.3 PostgreSQL readiness
  - [x] 0.4 Next.js static frontend
  - [x] 0.5 Typed API client
  - [x] 0.6 Go static serving
  - [x] 0.7 Goose/sqlc/env skeleton
  - [x] 0.8 OpenAPI/Makefile
  - [x] 0.9 Hardened deploy script
  - [x] 0.10 VPS provisioning
  - [x] 0.11 Remote deployment/smoke
  - [ ] 0.12 Delivery Gate
- [ ] Fase 1 — Database + admin authentication

## Commands

Pending scaffold. Expected remote verification families:

```bash
cd /home/ubuntu/informatika-monitoring/backend
go test ./... -count=1
go vet ./...
go build ./cmd/api

cd /home/ubuntu/informatika-monitoring/frontend
npm ci
npm run lint
npm run typecheck
npm run build
```

## Design mode

- antislop: DURING (session override for initial implementation).
- No generated logo/illustration/assets without separate approval.
- UI identity: warm Indonesian classroom progress board.
- Status always uses text + icon + color; Pengumpulan/Pemeriksaan/Buku Tulis stay separate.

## Current next step

Record Task 0.10–0.11 completion evidence, then execute Task 0.12 Antislop Delivery Gate and create `docs/F0_ACCEPTANCE.md`. Do not enter Fase 1 until the gate passes and the user approves the phase checkpoint.
