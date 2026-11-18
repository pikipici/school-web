# Fase 3B — Pengumpulan dan Buku Tulis: Acceptance

> Tanggal: 15 November 2026. Produksi: [fikriramaaulia.online](https://fikriramaaulia.online), rilis `20261118T045543Z-cffeaffbd4f3` (kode `cffeaff`, termasuk perbaikan hapus kelompok dan 360 px), migrasi versi 9.
> Status: **DITUTUP 18 Nov 2026 dengan pengecualian** atas instruksi user ("Tutup Fase 3B dengan pengecualian").

## Gerbang

| Gerbang | Hasil | Bukti |
|---|---|---|
| Migrasi bersih | PASS | Uji skema 3.6 (up/down, nomor urut, minimal satu berkas, mutan); produksi `OK 00009` setelah cadangan `pre-20261115T122908Z-f660abcc6378.dump` (35268 B) |
| AC02 PPT saja → Sebagian | PASS | Uji Go + E2E; produksi: pengumpulan awal PPT → "Terkumpul Sebagian" (sebelum revisi) |
| AC03 PDF menyusul → Lengkap | PASS | Uji Go + E2E; produksi: Revisi 1 PDF → "Berkas Digital Lengkap" di admin dan publik |
| AC08 revisi PPT tidak menghapus PDF | PASS | Uji Go + mutan + E2E (belum dicoba di produksi) |
| Klik ganda / nomor lompat → 409 | PASS | Uji Go + HTTP + E2E |
| Audit dalam transaksi yang sama | PASS | Audit gagal → perubahan batal (uji); produksi: 8 baris audit uji, tanpa email/subjek/catatan |
| Publik tanpa data privat | PASS | Respons produksi tanpa `gmail`, catatan, nama berkas, atau subjek |
| Admin wajib sesi + CSRF | PASS | Uji HTTP; produksi anonim 401 |
| Frontend uji, lint, typecheck, build | PASS | 108 uji Node; lint 0 |

## Cek browser produksi (Edge, 15 Nov 2026 19.31–19.38 WIB)

| Langkah | Hasil |
|---|---|
| Buat kelas `UJI`, Kelompok 1 (email format juknis) | Berhasil; tombol **Pengumpulan** membuka `/admin/kelompok/detail/?id=4` |
| Catat tanpa centang PPT/PDF | Ditolak di formulir: "Centang PPT, PDF, atau keduanya." |
| Pengumpulan awal: PPT + nama berkas + subjek juknis | Tercatat; waktu bawaan sekarang (WIB); pengirim bawaan email kelompok |
| Revisi 1: PDF + subjek `REVISI 1 - …` | Tercatat; lajur "Berkas Digital Lengkap"; tombol Hapus hanya pada putaran terakhir |
| Buku tulis diterima | "Sudah Diterima" di admin dan publik |
| Tabel publik | `LENGKAP` / `BELUM_DICEK` / `SUDAH_DITERIMA`, tanpa data privat |
| Hapus Revisi 1 lalu Pengumpulan awal (konfirmasi) | Berhasil berurutan; nomor berikutnya kembali "Pengumpulan awal" |
| Buku tulis dibatalkan | "Belum Diterima" |
| Bersih-bersih | 0 putaran; kelas `UJI` **dinonaktifkan** (lihat temuan 1); publik kosong |

## Temuan

1. **Kelompok tidak bisa dihapus setelah buku tulis pernah disimpan**, walau kemudian dibatalkan dan kosong: baris `group_notebooks` tetap ada dan `ON DELETE RESTRICT` menolak. **Diperbaiki** (15 Nov 2026): `DeleteGroup` kini membuang baris buku tulis yang "belum diterima, tanpa catatan" dalam transaksi yang sama sebelum menghapus kelompok; buku diterima, buku bercatatan, dan putaran tetap menolak (`IN_USE`), dan penolakan tidak menyentuh baris buku. RED mereproduksi kasus produksi; GREEN + seluruh suite 2 kali + `-race`; mutan "hapus semua baris buku" tertangkap. Data uji `UJI` dihapus setelah perbaikan dideploy (rilis `20261115T125117Z-a2d9fc52cbba`, cadangan 44271 B): Kelompok 1 dengan baris buku kosong terhapus lewat UI (audit `progress.group_deleted` 19.52.15), lalu kelas (`progress.class_deleted` 19.52.42); produksi 0 kelas, 0 kelompok, 0 buku, 0 putaran.

## Pengecualian

1. Keyboard halaman detail belum dicek. **360 px sudah terukur** (15 Nov 2026): Edge headless pribadi, lebar CSS 360, halaman `/admin/kelompok/detail/?id=1` dari produksi, dengan `/api/v1/*` dijawab JSON tiruan (nama kelompok 57 karakter, email 67 karakter, nama berkas 98 karakter tanpa spasi, subjek panjang, tautan panjang). Tidak login dan tidak menulis apa pun ke produksi. Hasil awal: **meluap**, `scrollWidth` 491 px; 10 elemen melewati tepi kanan (email, nama berkas, subjek, catatan, tombol Ubah). Penyebab: pada ≤ 720 px `.admin-row` memakai kolom `1fr`, yang tidak bisa menyusut di bawah kata terpanjang, dan teks meta tidak boleh patah. Perbaikan `77788ad`: kolom `minmax(0, 1fr)`, `min-width: 0` dan `overflow-wrap: anywhere` pada isi baris. Dengan CSS itu disuntikkan ke halaman yang sama: `scrollWidth` 360, 0 elemen meluap, baik tampilan biasa maupun saat formulir Ubah terbuka. Dideploy 18 Nov 2026 (rilis `20261118T045543Z-cffeaffbd4f3`, cadangan 43850 B, migrasi tetap 9, smoke PASS). Diukur ulang di produksi tanpa CSS suntikan: `scrollWidth` 360, 0 elemen meluap, tampilan biasa dan formulir Ubah. **360 px lulus.**
2. `datetime-local` di HP belum dicoba.
3. AC08 di produksi belum dicoba (hanya uji dan E2E).
4. Belum ada uji komponen UI.
5. Kunci `FOR UPDATE` di trigger nomor putaran tidak dibuktikan uji.
