# Monitoring Tugas Informatika Kelas IX

Website untuk memantau pengumpulan dan evaluasi tugas kelompok Informatika kelas IX.

## Prinsip produk

- Guru memeriksa Gmail dan berkas secara manual di luar aplikasi.
- Website hanya mencatat status, histori revisi, dan evaluasi yang dipublikasikan guru.
- Semua halaman pengunjung/siswa publik tanpa login, termasuk detail kelompok dan temuan terbit. **Login hanya untuk admin.** Tidak ada kode akses atau sesi kelompok.
- Draf, email/metadata pengumpulan, catatan internal, dan audit hanya untuk admin.
- Tidak ada integrasi Gmail, unggah tugas siswa, atau penilaian otomatis. Admin boleh mengunggah lampiran Materi PDF/PPT/PPTX.
- Backend dan seluruh aturan bisnis berada di Go; Next.js hanya frontend.

## Status

Fase 0, 1, 2A, 2B, 3A, dan 3B sudah ditutup; pengecualian tercatat di dokumen acceptance masing-masing. Pengumuman, Materi beserta lampiran, kelas/kelompok, pengumpulan manual, dan buku tulis sudah di produksi. Berikutnya **Fase 3C, Task 3.10–3.13**: temuan kesalahan terbit untuk publik tanpa login, UI admin/publik, deploy, dan panduan guru. Login tetap hanya admin. Dokumen sumber:

- `PLAN.md` — spesifikasi produk dan arsitektur.
- `DESIGN.md` — arah visual dan accessibility.
- `.kiro/steering/IMPLEMENTATION_PLAN.md` — backlog eksekusi Fase 0.
- `.kiro/steering/PHASE2_PLAN.md` — riwayat Fase 2A dan 2B.
- `.kiro/steering/PHASE3_PLAN.md` — backlog aktif Fase 3C dan kebijakan akses publik.
- `.kiro/steering/PROGRESS.md` — status aktual, bukti verifikasi, dan langkah berikutnya.
- `LOCAL_AI_CONTEXT.md` — pointer operasional sesi pengembangan.
- `docs/openapi.yaml` — kontrak endpoint Go yang sudah tersedia.
- `docs/DEPLOYMENT.md` — deployment boundary, pre-flight, rollback, dan smoke runbook.

## Kontrak API fondasi

Backend Go saat ini menyediakan:

- `GET /healthz` — liveness proses, independen dari database.
- `GET /readyz` — readiness database; `503` aman bila database belum siap.
- `GET /api/v1/public/summary` — ringkasan publik dari catatan nyata.
- `GET /api/v1/public/progress` — tabel progres publik tanpa login.

API publik detail kelompok dan temuan terbit direncanakan pada Task 3.10; belum tersedia. Autentikasi hanya untuk API admin.

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
