# Fase 2B — Lampiran Materi: Acceptance

> Tanggal: 14 November 2026. Produksi: [fikriramaaulia.online](https://fikriramaaulia.online), rilis `20261114T075621Z-d3918f34507a` (kode `d3918f3`), migrasi versi 7.
> Status: **SIAP DITUTUP dengan pengecualian** (lihat bagian akhir). Penutupan menunggu persetujuan user.

## Gerbang

| Gerbang (`PHASE2_PLAN.md`) | Hasil | Bukti |
|---|---|---|
| Validasi isi dengan berkas asli dan palsu; makro ditolak | PASS (kecuali `.ppt` asli) | Uji Go: PDF/PPT/PPTX sintetis, PPTX asli python-pptx, makro `vbaProject.bin`/`macroEnabled`/`VBA` ditolak, uji mutan. Produksi lewat browser: PDF asli (cetak Edge, 25.982 B) dan PPTX asli (28.356 B) diterima; `Tugas palsu.pdf` berisi HTML ditolak dengan pesan tetap |
| Batas ukuran, jumlah, kuota, sisa disk | PASS (uji) | Uji HTTP: 25 MiB + 1 B → 413, lampiran ke-6 → 409, kuota → 507, sisa disk → 507; Caddy uji nyata: 27 MB hanya jalur unggah, 1 MB lainnya |
| Unduhan hanya untuk materi terbit; header aman | PASS | Uji HTTP + uji mutan; produksi: materi draf → 404, setelah terbit → 200 dengan `attachment`, `nosniff`, `default-src 'none'; sandbox`; SHA-256 unduhan sama persis dengan berkas asli |
| Cadangan mencakup lampiran dan bisa dipulihkan | PASS | Uji fungsi deploy (89) termasuk pulihkan; deploy produksi menulis snapshot; `deploy.sh --check` di produksi merehearse snapshot 1 berkas; latihan pulih dari salinan hard link ke direktori sementara: berkas identik, mode 0640 |
| Deploy + smoke; cek browser dengan PDF dan PPTX asli | PASS | Deploy exit 0, smoke lulus; Edge nyata: unggah, daftar, terbit, unduh, hapus |
| User menyetujui fase berikutnya | MENUNGGU | — |

## Cek browser produksi (Edge, 14 Nov 2026 16.44–16.53 WIB)

| Langkah | Hasil |
|---|---|
| Buat materi uji (draf) | Bagian Lampiran muncul setelah disimpan |
| Unggah PDF lalu PPTX | Pesan sukses, daftar "2 dari 5", jenis, ukuran, waktu WIB |
| Unggah HTML bernama `.pdf` | Ditolak server: "Hanya berkas PDF, PPT, atau PPTX yang diterima, dan isinya harus sesuai." |
| Daftar publik saat draf | 404 |
| Terbitkan, buka halaman materi | Dua tautan unduh (`download`) dengan jenis dan ukuran |
| Unduh keduanya | 200, tipe benar, hash sama dengan berkas lokal |
| Hapus PPTX (konfirmasi di baris) | Hilang dari daftar dan dari disk |
| 360px: halaman materi dan editor admin | Tanpa geser samping (`scrollWidth` = 360), tautan unduh tinggi 44 px |
| Hapus materi uji | Baris, berkas di disk (0 tersisa), dan unduhan lama → 404 |
| Audit | `material_created`, 2× `attachment_uploaded`, `material_updated`, `attachment_deleted`, `material_deleted` |

Unggah berkas dilakukan dengan memasang berkas ke `<input type="file">` lewat DevTools (alat browser agen tidak bisa memilih berkas); tombol dan tampilan diklik serta diperiksa di Edge nyata.

## Pengecualian (belum terverifikasi)

1. **Berkas `.ppt` asli** belum diuji di mana pun; hanya `.ppt` sintetis di uji Go.
2. **Unggahan besar lewat produksi** (mis. 20–25 MB) belum dicoba; bilah progres dan tombol Batalkan tidak sempat terlihat karena berkas uji kecil. Tenggat 10 menit per rute belum teruji di server nyata.
3. **Keyboard dan zoom 200%** di bagian Lampiran belum diperiksa.
4. **Antivirus** tidak ada; JavaScript di dalam PDF tidak ditolak (risiko diterima di `PLAN.md` 0; diunduh sebagai `attachment` + `sandbox`).
5. **Snapshot di disk yang sama** dengan data; tidak melindungi dari kehilangan disk VPS.
6. Crash di antara tulis berkas dan sisip baris dapat meninggalkan berkas yatim; belum ada pembersih berkala.
7. Catatan desain: judul "Lampiran" di halaman publik memakai huruf biasa, bukan huruf judul; masuk daftar revisi desain.

## Temuan lain

- Proses uji lama `/tmp/informatika-monitoring-task06` (Task 0.6) masih mendengarkan di `*:18300`. Bukan produksi; belum dimatikan.
