# Informatika Monitoring — Local AI Context

> Pointer operasional. Update setelah setiap task/phase atau perubahan environment.

## TL;DR

- Produk: web pribadi guru berisi Pengumuman, Materi, dan monitoring/evaluasi manual tugas kelompok Informatika kelas IX.
- Website publik tanpa login pengunjung/siswa. Login hanya admin. Detail kelompok dan temuan terbit publik; draf, email/metadata pengumpulan, catatan internal, dan audit khusus admin. Tidak ada kode akses atau sesi kelompok.
- Backend wajib Go (`chi`, `pgxpool`, `sqlc`, Goose, PostgreSQL).
- Frontend wajib Next.js App Router + TypeScript + Tailwind, static export.
- Tidak ada Gmail API/IMAP, upload tugas siswa, atau AI grading. Upload PDF/PPT/PPTX hanya untuk lampiran Materi oleh admin pada Fase 2B.
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

> Checkpoint terbaru 18 Nov 2026: Task 3.10–3.12 sudah deployed, Task 3.13 menunggu acceptance operator dengan data nyata (panduan singkat di `docs/F3C_ACCEPTANCE.md`). Produksi aktif `20261118T122948Z-b340b3b1ca32`, kode `b340b3b`, migrasi 10. Root `/` = Pengumuman, `/pengumuman/` tetap alias, `/materi/` = Materi, `/progres/` = Progres Tugas. Menu: Pengumuman → Materi → Progres Tugas. Subheader semua tab dihapus; judul progres Progres Tugas Kelompok Kelas 9. Publik tanpa login siswa, login hanya admin. Riwayat lama di bawah bukan status runtime terkini; lihat `.kiro/steering/PROGRESS.md`.

- Fase 0 dan Fase 1 sudah ditutup pada 13 November 2026.
- Redesign web pribadi guru, HTTPS, HSTS satu hari, dan CSP `enforce` sudah aktif di produksi.
- Fase 2 sudah disetujui dan dibagi menjadi 2A (konten teks) lalu 2B (lampiran).
- Task 2.1 selesai di `43c81ba`; 2.2 `fb3c9be`; 2.3 `183770d`; 2.4 `c34e3e5` + `0f90555`; 2.5 `6498211` (frontend publik). Migrasi 4–6 dan kode Fase 2A belum masuk produksi.
- Backend exposes `/healthz`, safe `/readyz`, and honest zero-state `/api/v1/public/summary`; unknown API routes return JSON 404.
- Readiness uses a fake-Pinger-tested boundary and is verified against the real PostgreSQL on `cbt-vps` (`/readyz` returns database ok).
- GitHub `origin` configured and `main` pushed to `pikipici/school-web`.
- `cbt-vps` has a bare repo, clean Git checkout, least-privilege PostgreSQL role/database, `.env` mode 0600, matching hardened systemd unit, enabled service, immutable release, and verified port 8300 runtime at HEAD `e779d36`.
- Toolchain installed for remote verification: Go 1.27.4, Node 26.13.1 LTS, npm 11.20.0.
- VPS was previously cleaned of the old Crypto Station project.

## Phase tracker

- [x] Fase 0 — Foundation (ditutup 13 Nov 2026 dengan pengecualian)
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
  - [x] 0.12 Delivery Gate (pengecualian: cek browser nyata belum)
- [x] Fase 1 — Database + admin authentication (ditutup 13 Nov 2026; `.kiro/steering/PHASE1_PLAN.md`)
- [x] Fase 2A — Pengumuman dan Materi teks (ditutup 14 Nov 2026 dengan pengecualian)
- [x] Fase 2B — Lampiran PDF/PPT/PPTX (ditutup 14 Nov 2026 dengan pengecualian)
- [x] Fase 3A — Kelas, kelompok, progres publik (ditutup 15 Nov 2026 dengan pengecualian)
- [x] Fase 3B — Pengumpulan dan buku tulis (ditutup 18 Nov 2026 dengan pengecualian)
- [ ] Fase 3C — Temuan kesalahan publik tanpa login (Task 3.10–3.13)
- Fase 3D (kode akses/sesi kelompok) dan Fase 4 dibatalkan.

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

Berikutnya evaluasi UI terbaru dan acceptance alur admin → publik dengan data nyata (Task 3.13); jangan mengulang implementasi 3.10–3.12 yang sudah deployed. Catatan terbit tampil inline di `/progres/`; root `/` menampilkan Pengumuman. Jangan buat login siswa, kode akses, atau sesi kelompok. Produksi `https://fikriramaaulia.online`, release `20261118T122948Z-b340b3b1ca32`, migrasi 10. Uji/build/runtime hanya di VPS; integrasi memakai `/home/ubuntu/.informatika-monitoring-test.env`. Deploy/restart tetap perlu persetujuan eksplisit.
