# Monitoring Tugas Informatika Kelas IX

Website untuk memantau pengumpulan dan evaluasi tugas kelompok Informatika kelas IX.

## Prinsip produk

- Guru memeriksa Gmail dan berkas secara manual di luar aplikasi.
- Website hanya mencatat status, histori revisi, dan evaluasi yang dipublikasikan guru.
- Tidak ada integrasi Gmail, upload PPT/PDF, atau penilaian otomatis.
- Backend dan seluruh aturan bisnis berada di Go; Next.js hanya frontend.

## Status

Fase 0 foundation sedang berjalan. Task 0.1–0.8 sudah selesai: repository baseline, Go HTTP/readiness, Next.js static public foundation/client, static serving dari Go, Goose/sqlc/environment skeleton, serta kontrak OpenAPI dan perintah verifikasi terpadu. Dokumen sumber:

- `PLAN.md` — spesifikasi produk dan arsitektur.
- `DESIGN.md` — arah visual dan accessibility.
- `.kiro/steering/IMPLEMENTATION_PLAN.md` — backlog eksekusi Fase 0.
- `LOCAL_AI_CONTEXT.md` — pointer operasional sesi pengembangan.
- `docs/openapi.yaml` — kontrak endpoint Go yang sudah tersedia.

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

`make verify` bersifat fail-fast dan tidak pernah menjalankan deployment. Target `make deploy` hanya menjadi dispatcher eksplisit; target tersebut baru aktif setelah `deploy/deploy.sh` dibuat pada Task 0.9.
