# Fase 3A — Kelas, Kelompok, Tabel Publik: Acceptance

> Tanggal: 15 November 2026. Produksi: [fikriramaaulia.online](https://fikriramaaulia.online), rilis `20261115T002336Z-eff11c66953c` (kode `eff11c6`), migrasi versi 8.
> Status: **DITUTUP 15 Nov 2026 dengan pengecualian** atas persetujuan user.

## Gerbang

| Gerbang | Hasil | Bukti |
|---|---|---|
| Migrasi bersih, seed benar | PASS | Uji skema 3.1 (up/down, satu tahun aktif, unik tanpa beda huruf, mutan); produksi `OK 00008` setelah cadangan `pre-20261114T223049Z-0700f3215c1b.dump` |
| Go: uji, `-race`, vet, build, gofmt | PASS | 3.2–3.3 di `cbt-vps`; gerbang `deploy.sh` dua kali (rilis `0700f32` dan `eff11c6`) |
| Admin wajib sesi + CSRF + Origin; mutasi beraudit | PASS | Uji HTTP 401/403; produksi: `/api/v1/admin/classes` anonim 401; audit 6 baris uji tanpa email |
| Publik tanpa email/catatan (AC04 sebagian) | PASS | Uji kunci JSON persis + mutan; E2E 28 baris tanpa `email`/`@`; produksi menampilkan tanpa email |
| Kelompok baru tampil `Belum Mengumpulkan` (AC01) | PASS | E2E 4×7 di database uji; produksi 2 kelompok uji tampil `Belum Mengumpulkan` / `Belum Diperiksa` / `Belum Diterima` |
| Kelas/tahun ditambah tanpa ubah kode (AC20) | PASS | UI tambah tahun ajaran, aktifkan, tambah kelas (uji + produksi) |
| Frontend: uji, lint, typecheck, build | PASS | 102 uji Node; lint 0 masalah |
| Layar HP (AC18) | PASS setelah perbaikan | Lihat di bawah |
| OpenAPI | PASS (sebagian) | 0.5.0, 26 jalur, `$ref` lengkap; validator spesifikasi tidak terpasang di VPS |

## Cek browser produksi (Edge)

| Langkah | Hasil |
|---|---|
| Buat kelas `UJI` dan dua kelompok | Berhasil; tombol "Isi dengan format juknis" mengisi `kelompok1.uji.2627.informatika@gmail.com` |
| Tambah tanpa topik | Ditolak di formulir: "Pilih topik." |
| Nama ganda (`kelompok 1`) | Ditolak server: "Nama atau kode itu sudah dipakai…" |
| `/progres/` lebar penuh | Tabel tujuh kolom, status teks + ikon + warna |
| `/progres/` 360 px (rilis pertama) | **GAGAL**: lebar 404 px (kolom waktu tidak membungkus), teks topik lepas dari label, kata terpotong di tengah di desktop, petunjuk email keluar dari kotak |
| Perbaikan `eff11c6`, deploy ulang | 360 px: `scrollWidth` 360, topik satu baris grid; desktop: tidak ada kata terpotong |
| `/admin/kelas/`, `/admin/kelompok/` 360 px | Tanpa geser samping |
| Keyboard `/progres/` | Tab sampai kotak cari, ring fokus terlihat, ketik "panjang" → tabel tersaring ke satu baris |
| Bersih-bersih | Dua kelompok dan kelas `UJI` dihapus lewat UI (konfirmasi); produksi kembali 0 kelas, 0 kelompok |

## Pengecualian

1. Zoom 200% belum diperiksa.
2. Lebar 700 px dan 1000 px tidak terukur (skrip DevTools macet); hanya 360 px dan lebar penuh.
3. Tes komponen UI tidak ada; hanya logika klien.
4. Audit masih fail-open di luar transaksi (sama seperti 2A).
5. Validator OpenAPI tidak dijalankan.
6. Nama tahun ajaran baru belum diuji di produksi (hanya tambah kelas).
