# Fase 3B — Pengumpulan dan Buku Tulis: Acceptance

> Tanggal: 15 November 2026. Produksi: [fikriramaaulia.online](https://fikriramaaulia.online), rilis `20261115T122908Z-f660abcc6378` (kode `f660abc`), migrasi versi 9.
> Status: **SIAP DITUTUP dengan pengecualian.** Penutupan menunggu persetujuan user.

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

1. **Kelompok tidak bisa dihapus setelah buku tulis pernah disimpan**, walau kemudian dibatalkan dan kosong: baris `group_notebooks` tetap ada dan `ON DELETE RESTRICT` menolak. Akibatnya kelas `UJI` + Kelompok 1 (id 4) tertinggal di produksi dalam keadaan nonaktif. Usul perbaikan kecil: saat hapus kelompok, buang dulu baris buku tulis yang "belum diterima, tanpa catatan" dalam satu transaksi; baru setelah itu data uji bisa dihapus lewat UI.

## Pengecualian

1. 360 px dan keyboard halaman detail **belum terukur**: skrip DevTools macet, dan tangkapan headless tidak bisa sesempit 360 px. Halaman publik `/progres/` sudah lulus 360 px di 3A dan tata letaknya tidak berubah.
2. `datetime-local` di HP belum dicoba.
3. AC08 di produksi belum dicoba (hanya uji dan E2E).
4. Belum ada uji komponen UI.
5. Kunci `FOR UPDATE` di trigger nomor putaran tidak dibuktikan uji.
