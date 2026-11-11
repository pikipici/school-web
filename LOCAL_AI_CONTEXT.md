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
- GitHub remote: belum ditentukan

## Sources of truth

1. `PLAN.md` — product, data, API, security, acceptance AC01–AC25.
2. `DESIGN.md` — visual direction and antislop rules.
3. `.kiro/steering/IMPLEMENTATION_PLAN.md` — current execution checklist.
4. This file — environment and current checkpoint.

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
- Tasks 0.1–0.2 complete: repository baseline and tested Go HTTP foundation.
- Backend currently exposes `/healthz` and honest zero-state `/api/v1/public/summary`; unknown API routes return JSON 404.
- No GitHub remote configured.
- `cbt-vps` build-only workspace exists at `/home/ubuntu/informatika-monitoring`; no runtime, DB, `.env`, bare deploy repo, or systemd service exists yet.
- Toolchain installed for remote verification: Go 1.27.4, Node 26.13.1 LTS, npm 11.20.0.
- VPS was previously cleaned of the old Crypto Station project.

## Phase tracker

- [~] Fase 0 — Foundation
  - [x] 0.1 Repository hygiene
  - [x] 0.2 Go router foundation
  - [ ] 0.3 PostgreSQL readiness
  - [ ] 0.4 Next.js static frontend
  - [ ] 0.5 Typed API client
  - [ ] 0.6 Go static serving
  - [ ] 0.7 Goose/sqlc/env skeleton
  - [ ] 0.8 OpenAPI/Makefile
  - [ ] 0.9 Hardened deploy script
  - [ ] 0.10 VPS provisioning
  - [ ] 0.11 Remote deployment/smoke
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

Commit Task 0.2. Then run Task 0.3 strict TDD with a fake DB `Pinger`: safe `/readyz` success/failure behavior, graceful shutdown, and server hardening. Real database connectivity remains pending Task 0.10 approval; no DB, `.env`, bare deploy repo, systemd, port, or runtime provisioning yet.
