# Monitoring Tugas Informatika Kelas IX

Website untuk memantau pengumpulan dan evaluasi tugas kelompok Informatika kelas IX.

## Prinsip produk

- Guru memeriksa Gmail dan berkas secara manual di luar aplikasi.
- Website hanya mencatat status, histori revisi, dan evaluasi yang dipublikasikan guru.
- Tidak ada integrasi Gmail, upload PPT/PDF, atau penilaian otomatis.
- Backend dan seluruh aturan bisnis berada di Go; Next.js hanya frontend.

## Status

Fase 0 foundation sedang berjalan. Task 0.1–0.4 sudah selesai: repository baseline, Go HTTP/readiness foundation, dan Next.js static public foundation. Dokumen sumber:

- `PLAN.md` — spesifikasi produk dan arsitektur.
- `DESIGN.md` — arah visual dan accessibility.
- `.kiro/steering/IMPLEMENTATION_PLAN.md` — backlog eksekusi Fase 0.
- `LOCAL_AI_CONTEXT.md` — pointer operasional sesi pengembangan.

## Environment

Coding dan Git dilakukan pada Windows. Build, test, migrasi, dan runtime dilakukan di `cbt-vps`. Perintah setup akan dilengkapi setelah scaffold Fase 0 tersedia.
