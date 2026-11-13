# Monitoring Tugas Informatika Kelas IX

Website untuk memantau pengumpulan dan evaluasi tugas kelompok Informatika kelas IX.

## Prinsip produk

- Guru memeriksa Gmail dan berkas secara manual di luar aplikasi.
- Website hanya mencatat status, histori revisi, dan evaluasi yang dipublikasikan guru.
- Tidak ada integrasi Gmail, upload PPT/PDF, atau penilaian otomatis.
- Backend dan seluruh aturan bisnis berada di Go; Next.js hanya frontend.

## Status

Fase 0 dan Fase 1 sudah ditutup. Redesign web pribadi guru, HTTPS, HSTS, dan CSP sudah aktif di produksi. Fase aktif adalah **Fase 2A — Pengumuman dan Materi teks**: Task 2.1–2.3 sudah selesai (migrasi konten, paket Go `content`, dan API publik published-only). Berikutnya Task 2.4, API admin CRUD + audit. Perubahan Fase 2A belum dideploy ke produksi. Dokumen sumber:

- `PLAN.md` — spesifikasi produk dan arsitektur.
- `DESIGN.md` — arah visual dan accessibility.
- `.kiro/steering/IMPLEMENTATION_PLAN.md` — backlog eksekusi Fase 0.
- `.kiro/steering/PHASE2_PLAN.md` — backlog aktif Fase 2A dan 2B.
- `.kiro/steering/PROGRESS.md` — status aktual, bukti verifikasi, dan langkah berikutnya.
- `LOCAL_AI_CONTEXT.md` — pointer operasional sesi pengembangan.
- `docs/openapi.yaml` — kontrak endpoint Go yang sudah tersedia.
- `docs/DEPLOYMENT.md` — deployment boundary, pre-flight, rollback, dan smoke runbook.

## Kontrak API fondasi

Backend Go saat ini menyediakan:

- `GET /healthz` — liveness proses, independen dari database.
- `GET /readyz` — readiness database; `503` aman bila database belum siap.
- `GET /api/v1/public/summary` — ringkasan publik zero-state tanpa data palsu.

Semua route API yang tidak dikenal mengembalikan error JSON standar. Kontrak lengkap berada di `docs/openapi.yaml`.

## Verifikasi

Coding dan Git dilakukan pada Windows. Build, test, migrasi, dan runtime dilakukan di `cbt-vps`.

Dari root repository pada environment build yang dependency-nya sudah tersedia:

```bash
make test-backend
make vet-backend
make build-backend
make lint-frontend
make typecheck-frontend
make build-frontend
make verify
```

`make verify` bersifat fail-fast dan tidak pernah menjalankan deployment. `make deploy` merupakan dispatcher eksplisit ke `cbt-vps`; gunakan hanya dengan persetujuan eksplisit; provisioning Task 0.10 sudah selesai.
