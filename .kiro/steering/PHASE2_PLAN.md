# Fase 2 — Pengumuman, Materi, dan Lampiran (Rencana)

> Status: **DISETUJUI 13 Nov 2026 — 2A sedang dikerjakan (3/8 task selesai).**
> Dibuat: 13 November 2026. Sumber: `PLAN.md` bagian 0 (keputusan lampiran), `DESIGN.md` (Beranda dan navigasi), Fase 1 (login admin, CSRF, audit).

## Tujuan

Guru menulis Pengumuman dan Materi dari dashboard admin, dan melampirkan PDF/PPT/PPTX ke Materi. Pengunjung membaca dan mengunduh tanpa login. Beranda berhenti menampilkan keadaan kosong begitu konten ada.

## Usulan pembagian

Lampiran membawa risiko yang berbeda (disk, unggahan besar, cadangan berkas, isi berbahaya). Saya usulkan dua bagian yang masing-masing dideploy dan ditutup sendiri:

| Bagian | Isi | Risiko utama |
|---|---|---|
| **2A** | Pengumuman + Materi teks (tanpa lampiran) | XSS dari isi teks, bocornya draf ke publik |
| **2B** | Lampiran PDF/PPT/PPTX untuk Materi | disk penuh, unggahan besar, berkas berbahaya, cadangan |

## Di dalam scope

- Tabel `announcements`, `topics` (seed K1–K7 dari `PLAN.md` 2.1), `materials`, `material_attachments`.
- API admin CRUD (wajib login + CSRF + `Origin`) dan API publik baca-saja yang hanya memuat konten terbit.
- Halaman admin: daftar, buat, ubah, terbitkan/tarik, hapus (dengan konfirmasi).
- Halaman publik: daftar di Beranda, halaman daftar lengkap, dan halaman baca satu item.
- Audit untuk setiap perubahan konten.
- 2B: unggah, simpan di disk, unduh lewat Go, hapus, cadangan berkas.

## Di luar scope

- Kelas, kelompok, pengumpulan, evaluasi, kode akses kelompok (modul Progres Tugas; fase berikutnya).
- Komentar, notifikasi, email, RSS, pencarian teks penuh.
- Editor kaya (WYSIWYG) dan Markdown (lihat keputusan K2).
- Pemindai virus (dicatat sebagai risiko di `PLAN.md` bagian 0).

## Keputusan yang saya usulkan

| # | Hal | Usulan | Alasan |
|---|---|---|---|
| K1 | Draf dan terbit | Status `draft` / `published`; publik hanya melihat `published` | Guru bisa menyiapkan dulu; kebocoran draf dicegah di query Go, bukan di UI |
| K2 | Format isi | **Teks biasa**: baris kosong jadi paragraf, URL `https://` jadi tautan. Tanpa HTML/Markdown | Tidak ada jalur XSS; CSP tetap ketat |
| K3 | Pengumuman disematkan | Ya, satu bendera `pinned`; yang disematkan tampil paling atas | Pengumuman penting tidak tenggelam |
| K4 | Pengumuman kedaluwarsa | Tidak di 2A | Bisa ditambah nanti; menjaga sederhana |
| K5 | Materi per topik | Topik K1–K7 **opsional**; materi tanpa topik tampil di kelompok **Umum**; plus urutan | Web pribadi punya materi di luar tujuh topik (mis. panduan pengumpulan) |
| K6 | Hapus | Hapus permanen dengan konfirmasi; jejak tetap ada di `audit_log` (judul + id) | Konten publik, bukan data siswa; tidak perlu arsip |
| K7 | Alamat halaman | Halaman baca memakai `?id=` (mis. `/pengumuman/baca/?id=12`) | Ekspor statis Next.js tidak bisa membuat halaman per id saat runtime |
| K8 | Tipe ID | Tetap `bigint` seperti Fase 1, bukan UUID seperti tertulis di `PLAN.md` 9 | Konsisten dengan tabel yang sudah ada; `PLAN.md` diperbarui |
| K9 | Waktu | Simpan UTC, tampil WIB; `published_at` diisi saat pertama terbit | Sama dengan `DESIGN.md` |
| K10 | Batas panjang | Judul 1–160 karakter; isi pengumuman maks 10.000; deskripsi materi maks 20.000 | Mencegah isi raksasa; dipaksa di Go dan DB |
| K11 | Lampiran (2B) | Sesuai `PLAN.md` bagian 0: PDF/PPT/PPTX, cek isi, 25 MB, 5 per materi, total 2 GB, sisa disk min 1 GB | Sudah diputuskan |

## Temuan teknis yang memengaruhi rencana

1. **Batas Caddy 1 MB** di `deploy/Caddyfile` akan menolak unggahan 25 MB. 2B harus menaikkan batas hanya untuk jalur unggah.
2. **Timeout server Go 15 detik** (`ReadTimeout`/`WriteTimeout` di `cmd/api/main.go`). Unggah 25 MB di koneksi sekolah bisa lebih lama, dan unduhan juga. 2B butuh tenggat per-rute (`http.ResponseController`), bukan menaikkan seluruh server.
3. **RAM VPS sekitar 2 GB** (sekitar 1,4 GB tersedia). Unggahan harus dialirkan langsung ke berkas sementara, tidak pernah dibaca utuh ke memori; satu unggahan serentak sudah cukup.
4. **Disk VPS 25 GB kosong.** Kuota 2 GB aman, tetapi tetap dicek sebelum menyimpan.
5. **Cadangan saat ini hanya database.** Lampiran di `/home/ubuntu/informatika-monitoring-data/attachments/` perlu dicadangkan juga, dan perlu ruang (sampai 2 GB per salinan).
6. **CSP sudah `enforce`.** Halaman baru tetap aman selama tidak ada skrip inline baru di luar yang dibuat Next.js; hash dihitung otomatis per halaman.
7. **Unduhan lewat Go**, bukan file statis: `Content-Disposition: attachment`, `X-Content-Type-Options: nosniff`, dan tipe dari hasil validasi.

## Endpoint (garis besar)

| Method | Path | Siapa |
|---|---|---|
| GET | `/api/v1/public/announcements` | publik (terbit saja, terbaru dulu, berhalaman) |
| GET | `/api/v1/public/announcements/{id}` | publik |
| GET | `/api/v1/public/topics` | publik |
| GET | `/api/v1/public/materials` | publik (opsional `?topic=K1`) |
| GET | `/api/v1/public/materials/{id}` | publik |
| GET | `/api/v1/public/attachments/{id}` | publik, unduh (2B) |
| GET/POST | `/api/v1/admin/announcements` | admin |
| GET/PATCH/DELETE | `/api/v1/admin/announcements/{id}` | admin |
| GET/POST | `/api/v1/admin/materials` | admin |
| GET/PATCH/DELETE | `/api/v1/admin/materials/{id}` | admin |
| POST | `/api/v1/admin/materials/{id}/attachments` | admin, unggah (2B) |
| DELETE | `/api/v1/admin/attachments/{id}` | admin (2B) |

`PATCH` dengan `updatedAt` lama ditolak `409` (dua tab tidak saling menimpa diam-diam).

## Task

### 2A — Pengumuman dan Materi teks

| Task | Isi | Uji |
|---|---|---|
| 2.1 **DONE** | Migrasi `announcements`, `topics` + seed K1–K7, `materials`; query sqlc | uji skema: constraint, seed, up/down |
| 2.2 **DONE** | Paket `content`: validasi (panjang, status, topik), aturan terbit (`published_at`), konflik `updatedAt` | TDD unit + PostgreSQL nyata |
| 2.3 **DONE** | API publik baca-saja | TDD HTTP/PostgreSQL: draf tidak pernah bocor, 404 untuk draf/terhapus, paginasi |
| 2.4 | API admin CRUD + audit | TDD HTTP: 401 tanpa sesi, 403 tanpa CSRF, 409 konflik, audit tercatat |
| 2.5 | Frontend publik: Beranda memuat data, halaman daftar dan baca; perender teks biasa | uji klien API + uji perender (tidak ada HTML yang lolos) |
| 2.6 | Frontend admin: daftar, form, terbitkan/tarik, hapus dengan konfirmasi | lint/typecheck/build; keadaan loading/kosong/galat/sukses |
| 2.7 | OpenAPI, smoke baru, deploy, cek browser | `--check`, deploy, cek nyata |
| 2.8 | Gerbang 2A + `docs/F2A_ACCEPTANCE.md` | semua gerbang |

### 2B — Lampiran

| Task | Isi | Uji |
|---|---|---|
| 2.9 | Migrasi `material_attachments`; direktori data (`0750`, milik `ubuntu`, di luar rilis) | uji skema |
| 2.10 | Paket `attachments`: validasi isi (magic bytes, cek ZIP berisi `ppt/` untuk PPTX, OLE untuk PPT, tolak makro), nama UUID, tulis atomik (sementara lalu `rename`), kuota dan sisa disk | TDD dengan berkas contoh asli dan berkas palsu |
| 2.11 | Unggah dan hapus admin (dialirkan, tenggat per-rute, 1 unggahan serentak) | TDD HTTP: >25 MB 413, tipe palsu 415, kuota penuh 507, CSRF |
| 2.12 | Unduh publik lewat Go | TDD: header benar, hanya untuk materi terbit, Range jalan |
| 2.13 | Caddy: batas badan khusus jalur unggah; deploy: cadangan direktori lampiran | `caddy validate`, uji fungsi deploy |
| 2.14 | UI unggah (progres, galat jelas), daftar lampiran di halaman publik | lint/typecheck/build |
| 2.15 | Deploy, cek browser (unggah PDF dan PPTX asli, unduh), gerbang 2B + `docs/F2B_ACCEPTANCE.md` | semua gerbang |

Setiap task: uji gagal dulu, implementasi, uji/vet/build di VPS, commit.

## Risiko dan mitigasi

1. **Draf bocor ke publik.** Query publik selalu memfilter `status = 'published'`; uji khusus untuk setiap endpoint publik.
2. **XSS lewat isi.** Teks biasa saja; React meng-escape; perender tidak memakai `dangerouslySetInnerHTML`; CSP sebagai lapisan kedua.
3. **Berkas berbahaya.** Validasi isi, tolak makro, unduh sebagai `attachment` dengan `nosniff`; tanpa pemindai virus (risiko diterima, `PLAN.md` 0).
4. **Disk penuh atau VPS lambat saat unggah.** Kuota 2 GB, cek sisa disk, satu unggahan serentak, dialirkan ke disk.
5. **Berkas yatim** (berkas tanpa baris DB atau sebaliknya). Tulis berkas dulu, lalu baris DB dalam transaksi; hapus baris lalu berkas; pembersih berkala mencatat sisa.
6. **Cadangan tidak lengkap.** 2B wajib menambah cadangan direktori lampiran sebelum dianggap selesai.
7. **Migrasi maju-saja.** Cadangan `pg_dump` otomatis dari Fase 1 tetap berjalan sebelum migrasi.

## Keputusan user (13 Nov 2026)

User mengikuti semua rekomendasi: dibagi 2A lalu 2B; isi teks biasa; Pengumuman punya draf/terbit dan sematkan; hapus permanen dengan jejak audit; topik Materi opsional (tanpa topik = Umum). ID tetap `bigint` (K8).

## Gerbang selesai 2A

- [ ] Migrasi up/down bersih; seed topik benar.
- [ ] Uji Go (unit + integrasi) lulus, juga `-race`; vet, build, gofmt, sqlc hijau.
- [ ] Draf tidak pernah muncul di endpoint publik (terbukti uji).
- [ ] Semua aksi admin wajib sesi + CSRF + Origin; semua tercatat di audit.
- [ ] Frontend lint, typecheck, test, build; tidak ada HTML dari isi yang dirender.
- [ ] OpenAPI diperbarui.
- [ ] Deploy + smoke lulus; cek browser nyata (buat, terbitkan, lihat publik, tarik, hapus).
- [ ] User menyetujui lanjut ke 2B.

## Gerbang selesai 2B

- [ ] Validasi isi terbukti dengan berkas asli dan palsu; makro ditolak.
- [ ] Batas ukuran, jumlah, kuota, dan sisa disk terbukti uji.
- [ ] Unduhan hanya untuk materi terbit; header aman.
- [ ] Cadangan mencakup direktori lampiran dan terbukti bisa dipulihkan.
- [ ] Deploy + smoke lulus; cek browser nyata dengan PDF dan PPTX asli.
- [ ] User menyetujui fase berikutnya.

## Catatan pelaksanaan

- **Task 2.1 selesai (13 Nov 2026).** RED: 5 uji skema baru gagal tanpa migrasi. GREEN di ``cbt-vps`` dengan database uji: migrasi ``00004_create_announcements``, ``00005_create_topics`` (seed K1–K7, judul persis dari ``PLAN.md`` 2.1), ``00006_create_materials`` (topik opsional, ``ON DELETE RESTRICT``). Uji membuktikan: default draf/tidak disematkan, batas judul 160 dan isi 10.000/20.000 karakter (dihitung per karakter, termasuk huruf beraksen), status hanya draft/published, terbit wajib punya ``published_at``, topik tak dikenal ditolak, topik yang dipakai tidak bisa dihapus, ``down-to 3`` menghapus tabel konten tanpa menyentuh data Fase 1, ``up`` ulang tidak menggandakan seed. ``goose validate``, ``sqlc generate``, seluruh suite, vet, build hijau.
  - **Gladi migrasi pada data produksi:** cadangan produksi terbaru (``pre-20261113T023752Z-56c1e059e84d.dump``) dipulihkan ke database **uji**, lalu ``goose up``: versi 3 ke 6, admin 1 dan audit 6 tetap, topik 7. Database uji dikosongkan lagi sesudahnya. Catatan: selama gladi, database uji sempat berisi salinan data produksi (hash sandi admin dan audit).
  - **Temuan:** ``backend/cmd/api/main.go`` (dari commit CSP ``3f0361f``, sudah dideploy) tidak lolos ``gofmt`` (kurang baris baru di akhir). Diperbaiki, dan ``deploy.sh`` kini menolak deploy bila ada berkas Go yang belum di-``gofmt``.
  - Query sqlc untuk konten ditulis bersama task yang memakainya (2.2–2.4).
- **Task 2.2 selesai (13 Nov 2026, commit `fb3c9be`).** Strict RED di `cbt-vps`: paket uji gagal dibangun karena API yang diinginkan (`Input`, `Kind`, `Draft`, `Announcement`) belum ada. GREEN: paket `internal/content` dan query sqlc memvalidasi judul Unicode 1–160 setelah trim, isi Pengumuman 10.000, deskripsi Materi 20.000, status draf/terbit, topik opsional yang harus ada, dan urutan nonnegatif. Terbit pertama mengisi `published_at`; tarik lalu terbitkan ulang mempertahankan waktu pertama. Update dengan `updatedAt` lama menghasilkan `ErrConflict`, sedangkan ID tidak ada menghasilkan `ErrNotFound`. `sqlc generate`, targeted/full suite dua kali, race package, vet, build, dan gofmt hijau dengan PostgreSQL uji nyata. Belum di produksi.
- **Task 2.3 selesai (13 Nov 2026, commit `183770d`).** Lima endpoint publik read-only ditambahkan untuk daftar/detail Pengumuman, daftar Topik, serta daftar/detail Materi. Query publik selalu memfilter `published`; draf dan ID tidak ada menghasilkan JSON 404 identik. Daftar Pengumuman diurutkan sematan lalu waktu terbit; Materi mendukung filter kode topik dan urutan topik/materi. Paginasi `page`/`limit` memakai default 1/20, maksimum 100, serta menolak nilai cacat/overflow. Respons camelCase tidak memuat status, sematan, atau metadata internal. Tes ditulis sebelum implementasi; RED remote pertama tertahan bootstrap karena output sqlc memang tidak dilacak, sehingga bukti itu tidak diklaim sebagai RED perilaku. Setelah `sqlc generate`, targeted PostgreSQL, full suite dua kali, race paket terkait, vet, build `api`/`adminctl`, gofmt, dan diff check hijau. Belum di produksi.
