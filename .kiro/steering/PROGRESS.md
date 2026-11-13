# Informatika Monitoring — Progress

> Dokumen status aktual implementasi. Update setelah setiap task, verifikasi, commit, provisioning, dan deployment.
>
> Last updated: 14 November 2026, setelah Task 2.6

## Ringkasan

- Fase aktif: **Fase 2A — Pengumuman dan Materi teks**. Fase 0 dan Fase 1 sudah ditutup.
- Progress Fase 2A: **6/8 task selesai** (2.1–2.6). Fase 2B (lampiran): 0/7, belum mulai.
- Task terakhir selesai: **2.6 — Frontend admin**
- Next task: **2.7 — OpenAPI, smoke, deploy, cek browser** (butuh izin deploy; migrasi 4–6 masuk produksi)
- Local branch: `main`, working tree bersih
- Semua commit sampai `21532e1` sudah di `server/main`; commit Task 2.6 belum di-push.
- Produksi: **`https://fikriramaaulia.online`** aktif (Caddy + Let's Encrypt, berlaku s.d. 2027-02-11)
- Rilis produksi: `20261113T023752Z-56c1e059e84d` (kode `56c1e05`), migrasi versi **3**. Migrasi 4–6 dari Task 2.1 **belum** di produksi.
- PostgreSQL aplikasi: aktif, readiness PASS
- Admin: satu akun `fikri`; login dan logout lewat browser terbukti di audit

## Phase Tracker

| Fase | Isi | Status | Rincian |
|---|---|---|---|
| 0 | Fondasi Go + Next.js + deploy | DITUTUP 13 Nov 2026 (dengan pengecualian) | `.kiro/steering/IMPLEMENTATION_PLAN.md`, `docs/F0_ACCEPTANCE.md` |
| Redesign | Web pribadi guru (Pengumuman, Materi, Progres Tugas) | DONE, ter-deploy | `.kiro/steering/REDESIGN_BACKLOG.md` |
| 1 | Database + login admin + HTTPS | DITUTUP 13 Nov 2026 | `.kiro/steering/PHASE1_PLAN.md`, `docs/F1_ACCEPTANCE.md` |
| Hardening | HSTS 1 hari + CSP `enforce` | DONE, aktif di produksi | bagian Current Next Step |
| 2A | Pengumuman + Materi teks | 6/8 — berjalan | `.kiro/steering/PHASE2_PLAN.md` |
| 2B | Lampiran PDF/PPT/PPTX | belum mulai | `.kiro/steering/PHASE2_PLAN.md` |

## Fase 2A Tracker

| Task | Scope | Status | Commit | Verification |
|---|---|---|---|---|
| 2.1 | Migrasi `announcements`, `topics` + seed K1–K7, `materials` | DONE | `43c81ba` | RED 5 uji skema; GREEN di `cbt-vps`; `goose validate`, sqlc, seluruh suite, vet, build hijau; gladi migrasi 3→6 pada salinan data produksi di database uji (admin dan audit utuh) |
| 2.2 | Paket `content`: validasi, terbit, konflik | DONE | `fb3c9be` | Strict RED (`Input`, `Kind`, `Draft`, `Announcement` belum ada); sqlc; targeted + full suite 2x; race; vet; build; gofmt hijau di `cbt-vps` dengan PostgreSQL uji nyata |
| 2.3 | API publik baca-saja | DONE | `183770d` | Tes ditulis lebih dulu; remote RED awal tertahan bootstrap karena output sqlc sengaja tidak dilacak, lalu sqlc generate; targeted PostgreSQL, full suite 2x, race, vet, dua binary, gofmt hijau |
| 2.4 | API admin CRUD + audit | DONE | `c34e3e5`, `0f90555` | CRUD real-router + PostgreSQL; 401/403/409/422; CSRF/Origin; audit aman; full suite 2x, race paket terkait, vet, dua binary, gofmt hijau di `cbt-vps` |
| 2.5 | Frontend publik | DONE | `6498211` | 43 Node tests; lint/typecheck; static build 2x; lima output route; production audit 0; query-ID dan plain-text/link safety teruji di `cbt-vps` |
| 2.6 | Frontend admin | DONE | lihat commit Task 2.6 | 34 uji klien admin baru (RED dulu: modul belum ada); total 77 uji frontend; lint, typecheck, build (rute `/admin/pengumuman/ubah/`, `/admin/materi/ubah/`), audit produksi 0 di `cbt-vps`; E2E klien asli melawan API Go asli + database uji: buat, terbit, publik melihat, konflik 409, ubah, tarik, CSRF salah 403, hapus, 401 setelah logout; audit tercatat. Cek browser nyata belum (ikut Task 2.7) |
| 2.7 | OpenAPI, smoke, deploy, cek browser | PENDING | — | — |
| 2.8 | Gerbang 2A + `docs/F2A_ACCEPTANCE.md` | PENDING | — | — |

Keputusan Fase 2 (disetujui user 13 Nov 2026): dibagi 2A lalu 2B; isi teks biasa (tanpa Markdown/HTML); Pengumuman punya draf/terbit dan sematkan; hapus permanen dengan jejak audit; topik Materi opsional (tanpa topik = Umum); ID tetap `bigint`.

## Sources of Truth

1. `PLAN.md` — spesifikasi produk, arsitektur, security, API, dan AC01–AC25.
2. `DESIGN.md` — design tokens, UX, responsive, accessibility, dan antislop rules.
3. `.kiro/steering/IMPLEMENTATION_PLAN.md` (Fase 0), `.kiro/steering/PHASE1_PLAN.md` (Fase 1), `.kiro/steering/PHASE2_PLAN.md` (Fase 2) — rencana task-by-task.
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
| 0.9 | Hardened LMS-style deploy script | DONE | `a0bfb2f` | Shell syntax, isolated remote `--check`, all build gates, non-mutation assertions, and independent re-review PASS |
| 0.10 | Provision `cbt-vps` runtime workspace | DONE | server state | Bare repo, clean checkout, least-privilege DB/role, `.env` 0600, matching hardened unit, enabled service, and port ownership verified |
| 0.11 | Remote baseline deployment dan smoke | DONE | deployed `e779d36` | Full remote deploy gates, immutable release, systemd, health/readiness, frontend smoke, clean tree, and independent evidence review PASS |
| 0.12 | Antislop Delivery Gate dan phase close | DONE (dengan pengecualian) | `8300336` deployed | Gate otomatis PASS; cek browser nyata TIDAK diverifikasi; lihat `docs/F0_ACCEPTANCE.md` |

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
- Host: `43.134.111.212` (Tencent Cloud)
- User: `ubuntu`
- Git checkout/runtime workspace: `/home/ubuntu/informatika-monitoring`
- Bare deploy repository: `/home/ubuntu/git-repos/informatika-monitoring.git` (remote `server` di lokal; di checkout VPS bernama `origin`)
- Service: `informatika-monitoring-api.service`, port internal `8300` (tertutup dari internet)
- Reverse proxy: Caddy v2.11.9 (`caddy.service`, aktif dan enabled), konfigurasi `/etc/caddy/Caddyfile` = `deploy/Caddyfile`

Verified toolchain:

```text
Go:      1.27.4
Node.js: 26.13.1 LTS
npm:     11.20.0
Caddy:   2.11.9
```

Current provisioning state:

- Database `informatika_monitoring` (versi migrasi 3) dan role aplikasi non-superuser: PRESENT
- Database uji `informatika_monitoring_test` + role sendiri (tanpa akses produksi): PRESENT; kredensial `/home/ubuntu/.informatika-monitoring-test.env` (0600)
- `.env` produksi (0600, `ubuntu`): `APP_ADDR`, `PUBLIC_DIR`, `DATABASE_URL`, `ADMIN_COOKIE_SECURE=true`, `CSP_MODE=enforce`
- Cadangan otomatis sebelum migrasi: `/home/ubuntu/informatika-monitoring/backups/` (10 terakhir)
- DNS `fikriramaaulia.online`: A ke `43.134.111.212`, Cloudflare **DNS only** (wajib tetap abu-abu agar sertifikat bisa diperpanjang)
- Firewall Tencent: TCP 80 dan 443 terbuka; 8300 dan 5432 tidak terjangkau dari luar (terverifikasi)
- HSTS `max-age=86400`; cadangan Caddyfile lama `/etc/caddy/Caddyfile.bak-hsts3600`

## Current Backend Contract

Di produksi (`docs/openapi.yaml` 0.2.0):

- `GET /healthz`, `GET /readyz`, `GET /api/v1/public/summary` (zero state).
- `POST /api/v1/admin/auth/login`, `POST /api/v1/admin/auth/logout`, `GET /api/v1/admin/auth/me` — sesi server-side, cookie `__Host-admin_session`, CSRF + Origin, pembatas percobaan, audit.
- Go menyajikan ekspor statis Next.js dengan Content-Security-Policy berbasis hash per halaman.
- CLI server `bin/current/adminctl` (`create-admin`, `reset-password`).

Di kode, belum di produksi:

- Tabel `announcements`, `topics` (K1–K7), `materials` (Task 2.1).
- Paket Go `content` + query sqlc untuk validasi, create/update, aturan terbit pertama, topik opsional, dan optimistic concurrency (Task 2.2).
- API publik Pengumuman/Topik/Materi: published-only, detail aman, filter topik, dan paginasi berbatas (Task 2.3).
- API admin CRUD Pengumuman/Materi dengan sesi, CSRF, Origin, optimistic concurrency, hapus permanen, dan audit aman (Task 2.4).
- Frontend publik Beranda, daftar/detail Pengumuman dan Materi, filter topik, paginasi, serta perender teks biasa aman (Task 2.5).

Belum ada:

- UI admin Pengumuman/Materi (Task 2.6).
- Lampiran (Fase 2B).
- Kelas, kelompok, pengumpulan, evaluasi, portal kelompok.

## Safety and Scope Status

Tetap berlaku:

- Tanpa Gmail API/IMAP/sinkronisasi inbox, tanpa AI grading, tanpa data siswa asli.
- Tanpa upload dari siswa. Upload hanya untuk lampiran Materi dari guru (Fase 2B, belum dibangun).
- Frontend tidak mengakses database; tanpa Route Handlers/Server Actions Next.js.
- Tidak ada deploy/restart produksi tanpa persetujuan user.
- Agen tidak bisa memperbarui `main` di remote; itu dilakukan user.

## Blockers / Pending External Decisions

1. Tidak ada blocker untuk Task 2.6.
2. Docker tidak tersedia di `cbt-vps`; uji integrasi memakai database uji PostgreSQL lokal.
3. 7 advisory npm moderate (alat dev saja) masih terbuka sejak Fase 0.
4. HSTS 1 tahun menunggu kira-kira dua minggu HTTPS stabil (sekitar 27 Nov 2026).
5. Komponen UI belum punya uji otomatis; hanya logika klien API.

## Task 0.10–0.11 Deployment Evidence

Verified on `cbt-vps`:

```text
checkout/deployed HEAD        e779d364ec240941d95fa0137a37ba4ca5728aa9
systemd unit hash             matches repository template
service active/enabled        PASS
MainPID owns port 8300        PASS
service cgroup                /system.slice/informatika-monitoring-api.service
PostgreSQL readiness          PASS
/healthz                      {"status":"ok"}
/readyz                       {"status":"ok","database":"ok"}
frontend marker + smoke       PASS
.env ownership/mode           ubuntu:ubuntu 0600
DB owner                      informatika_monitoring_app
DB elevated role flags        false:false:false
server working tree           CLEAN
recent error journal          no entries
immutable release             20261112T140834Z-e779d364ec24
independent evidence review   VERIFIED, no acceptance blocker
```

The remote deploy command passed Go test/vet/build, `sqlc generate`, frontend `npm ci`/lint/typecheck/build, dependency readiness, immutable release switching, restart, readiness polling, and the checked-in smoke script. There are currently no Goose migration files, so connectivity was verified and migration execution was explicitly skipped.

## Current Next Step

Task 2.5 selesai. Saat dilanjutkan:

1. Sinkronkan commit lokal yang belum ada di `origin`/`server` sesuai workflow project.
2. Task 2.6 — frontend admin: daftar/form Pengumuman dan Materi, terbitkan/tarik, hapus dengan konfirmasi, serta state loading/kosong/galat/sukses.
3. Lanjut 2.7–2.8 sesuai `.kiro/steering/PHASE2_PLAN.md`. Migrasi 4–6 baru masuk produksi saat deploy Task 2.7 (cadangan otomatis berjalan sebelumnya).

Selesai sebelum jeda (13 Nov 2026):

- Fase 1 ditutup; HTTPS aktif; login/logout lewat browser terbukti di audit.
- HSTS dinaikkan ke 1 hari (`max-age=86400`); naikkan ke 1 tahun sekitar 27 Nov 2026.
- CSP `enforce` aktif (rilis `20261113T023752Z-56c1e059e84d`, kode `3f0361f`): terbukti tidak memutus `/progres/`, pengalihan `/admin/` ke login, dan login penuh (audit `auth.login` sukses 09:47:10 WIB dari `202.152.195.111`). Batas: tidak terbukti bahwa alat konsol menangkap laporan CSP. Undo: `CSP_MODE=off` lalu restart.
- Rencana Fase 2 disetujui.
- Task 2.1 selesai; `deploy.sh` kini menolak berkas Go yang belum di-`gofmt` (temuan: `cmd/api/main.go` dari commit CSP; sudah diperbaiki).
- Task 2.2 selesai (`fb3c9be`): paket `content` memvalidasi Unicode/panjang/status/topik/urutan, menjaga waktu terbit pertama, dan membedakan konflik `updatedAt` dari data tidak ditemukan. Verifikasi PostgreSQL nyata, race, vet, dan build hijau; belum di produksi.
- Task 2.3 selesai (`183770d`): lima endpoint publik read-only hanya mengekspos konten terbit, menyamakan 404 draf/tidak ada, menyediakan filter topik dan paginasi berbatas, serta terpasang hanya ketika database tersedia. Belum di produksi.
- Task 2.4 selesai (`c34e3e5`, hardening `0f90555`): CRUD admin real-router dilindungi sesi/CSRF/Origin, konflik edit 409, validasi topik 422, dan enam aksi audit teruji. Catatan: audit memakai recorder fail-open setelah mutasi sehingga mutation+audit belum atomik; kegagalan audit dapat meninggalkan mutasi tanpa jejak. Belum di produksi.
- Task 2.5 selesai (`6498211`): Beranda dan route statis daftar/detail Pengumuman/Materi memakai API relatif, detail runtime lewat `?id=`, filter topik, paginasi, WIB, dan perender teks biasa yang hanya menautkan HTTP(S). Build statis dan audit produksi hijau; browser automation viewport belum tersedia. Belum di produksi.

## Update Rule

After each task:

1. Update the tracker row and summary percentage.
2. Record RED evidence for behavior implemented with TDD.
3. Record exact remote verification commands/results.
4. Record independent review blockers and final verdict.
5. Record commit hash.
6. Advance `Current Next Step`.
7. Never mark live DB/deploy/manual acceptance as PASS unless directly verified.
