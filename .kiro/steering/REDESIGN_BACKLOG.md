# Redesign Backlog — Web Pribadi Guru

> Status: **DRAFT — menunggu approval. Belum ada kode diubah.**
> Dibuat: 13 November 2026.

## Konteks

Fase 0 membangun "Papan Progres Informatika" sebagai alat tunggal untuk monitoring tugas kelompok. Arah sebenarnya: **web pribadi guru** yang memuat konten sekolah. Progres tugas kelompok hanya salah satu bagian, bukan seluruh produk. Tampilan Fase 0 terasa seperti alat sekolah, bukan rumah pribadi, jadi belum sesuai harapan.

## Keputusan user (13 Nov 2026)

| Hal | Keputusan |
|---|---|
| Halaman depan | Langsung konten, tanpa hero perkenalan panjang |
| Konten | Pengumuman, Materi, Progres Tugas Kelompok |
| Nuansa | Rapi-profesional |
| Acuan | Mengikuti rekomendasi: [leerob.com](https://leerob.com) (daftar konten bersih) dan [gov.uk](https://www.gov.uk) (pengumuman jelas, mudah dibaca) |

## Perilaku yang diharapkan

1. Header: nama guru + menu kecil (Pengumuman, Materi, Progres Tugas).
2. Beranda: pengumuman terbaru di depan, lalu daftar materi.
3. "Progres Tugas Kelompok" menjadi satu menu/halaman sendiri; jalur Pengumpulan, Pemeriksaan, Buku Tulis tetap terpisah di dalamnya.
4. Peringatan "pembaruan manual" tetap ada, tapi hanya di halaman progres.
5. Tidak ada data, statistik, atau testimoni palsu.

## Dampak pada dokumen

| Dokumen | Perubahan |
|---|---|
| `PLAN.md` | Ruang lingkup produk melebar: tambah Pengumuman dan Materi. Putuskan apakah masuk MVP atau fase baru. |
| `DESIGN.md` | Overview, Layout, dan Colors perlu ditulis ulang. Tema "papan kendali kelas" diganti arah rapi-profesional. |
| `.kiro/steering/IMPLEMENTATION_PLAN.md` | Fase 0 sudah selesai di fondasi; redesign jadi task terpisah. |
| `docs/openapi.yaml` | Tambah endpoint pengumuman/materi hanya jika disetujui. |

## Jawaban user (13 Nov 2026)

1. Pengumuman dan Materi ditulis lewat **dashboard admin** (butuh login admin, tabel, dan API Go).
2. Materi boleh **lampiran file dari guru**. Ini scope baru: aturan lama hanya melarang upload PPT/PDF dari siswa. Perlu aturan tipe file, ukuran maksimal, dan tempat simpan.
3. Nama web: **nama pribadi** (nama persisnya belum diberikan).
4. Tampil **nama saja**, tanpa foto.
5. **Redesign dulu**, baru Fase 0 ditutup.

## Pertanyaan terbuka

1. Nama pribadi yang dipakai di header.
2. Lampiran: tipe file apa (PDF, PPT, DOCX, gambar), ukuran maksimal, dan simpan di disk VPS atau layanan lain?
3. Redesign tahap pertama hanya tampilan (kerangka + keadaan kosong jujur). Pengumuman dan Materi terisi baru setelah login admin dan database (Fase 1+). Setuju?

## Acceptance criteria

- [ ] `PLAN.md` dan `DESIGN.md` direvisi dan disetujui user.
- [ ] Beranda menampilkan konten, bukan perkenalan.
- [ ] Menu hanya berisi halaman yang ada.
- [ ] Lolos ulang gate kontras, 360px, zoom 200%, dan keyboard.
- [ ] Deploy ke `cbt-vps` dan smoke lulus.

## Status

| Langkah | Status |
|---|---|
| Arah desain diputuskan | DONE |
| Jawab 5 pertanyaan awal | DONE |
| Jawab 3 pertanyaan terbuka baru | DONE (nama sementara, lampiran PDF+PPT, redesign tahap satu hanya tampilan) |
| Revisi `DESIGN.md` | DONE (13 Nov 2026) |
| Revisi `PLAN.md` | DONE (v2.1, bagian 0; aturan lampiran ukuran/penyimpanan masih terbuka) |
| Implementasi tampilan tahap satu | DONE di working tree, belum commit/deploy. Lint, typecheck, build PASS di `cbt-vps` (salinan sementara, runtime tidak disentuh). Rute `/` dan `/progres` terlayani Go. |
| Cek browser manual (360px, zoom 200%, keyboard) | PENDING |
| Nama asli menggantikan "Guru Informatika" di `frontend/src/lib/site.ts` | PENDING |
