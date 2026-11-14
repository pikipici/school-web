# Fase 2 — Pengumuman, Materi, dan Lampiran (Rencana)

> Status: **DISETUJUI 13 Nov 2026 — 2A ditutup 14 Nov 2026; 2B berjalan (6/7: 2.9–2.14).**
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
| 2.4 **DONE** | API admin CRUD + audit | TDD HTTP/PostgreSQL: 401 tanpa sesi, 403 tanpa CSRF, 409 konflik, audit tercatat |
| 2.5 **DONE** | Frontend publik: Beranda memuat data, halaman daftar dan baca; perender teks biasa | uji klien API + uji perender (tidak ada HTML yang lolos) |
| 2.6 **DONE** | Frontend admin: daftar, form, terbitkan/tarik, hapus dengan konfirmasi | lint/typecheck/build; keadaan loading/kosong/galat/sukses |
| 2.7 | OpenAPI, smoke baru, deploy, cek browser | `--check`, deploy, cek nyata |
| 2.8 **DONE** | Gerbang 2A + `docs/F2A_ACCEPTANCE.md` | semua gerbang |

### 2B — Lampiran

| Task | Isi | Uji |
|---|---|---|
| 2.9 **DONE** | Migrasi `material_attachments`; direktori data (`0750`, milik `ubuntu`, di luar rilis) | uji skema |
| 2.10 **DONE** | Paket `attachments`: validasi isi (magic bytes, cek ZIP berisi `ppt/` untuk PPTX, OLE untuk PPT, tolak makro), nama UUID, tulis atomik (sementara lalu `rename`), kuota dan sisa disk | TDD dengan berkas contoh asli dan berkas palsu |
| 2.11 **DONE** | Unggah dan hapus admin (dialirkan, tenggat per-rute, 1 unggahan serentak) | TDD HTTP: >25 MB 413, tipe palsu 415, kuota penuh 507, CSRF |
| 2.12 **DONE** | Unduh publik lewat Go | TDD: header benar, hanya untuk materi terbit, Range jalan |
| 2.13 **DONE, aktif di produksi 14 Nov 2026** | Caddy: batas badan khusus jalur unggah; deploy: cadangan direktori lampiran | `caddy validate`, uji fungsi deploy |
| 2.14 **DONE** | UI unggah (progres, galat jelas), daftar lampiran di halaman publik | lint/typecheck/build |
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

- [x] Migrasi up/down bersih; seed topik benar.
- [x] Uji Go (unit + integrasi) lulus, juga `-race`; vet, build, gofmt, sqlc hijau.
- [x] Draf tidak pernah muncul di endpoint publik (terbukti uji).
- [x] Semua aksi admin wajib sesi + CSRF + Origin; semua tercatat di audit.
- [x] Frontend lint, typecheck, test, build; tidak ada HTML dari isi yang dirender.
- [x] OpenAPI diperbarui.
- [x] Deploy + smoke lulus; cek browser nyata (buat, terbitkan, lihat publik, tarik, hapus). Pengecualian: admin di 360px tidak diverifikasi.
- [x] User menyetujui lanjut ke 2B (14 Nov 2026).

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
- **Task 2.4 selesai (13 Nov 2026, commit `c34e3e5`, hardening `0f90555`).** CRUD admin Pengumuman/Materi tersedia untuk daftar, detail, buat, ubah dengan `updatedAt`, dan hapus permanen. Semua route memakai `AdminAuth.Require`; mutasi wajib Origin dan CSRF. POST menghasilkan 201 + Location, PATCH 200, DELETE 204; konflik 409, topik invalid 422, dan input/body/media type ditolak aman. Integrasi real-router PostgreSQL membuktikan route tanpa trailing slash tidak redirect, 401/403, lifecycle kedua jenis konten, konflik, serta enam audit event dengan actor dan detail terbatas `entityId`/`title`; mutasi gagal tidak menambah audit. RED hardening menemukan route koleksi tanpa slash dan fixture waktu yang perlu maju untuk membuktikan stale update. Full suite dua kali, race paket HTTP/content/audit/auth, vet, dua binary, gofmt, dan diff check hijau di `cbt-vps`. **Batas:** recorder audit tetap fail-open dan dipanggil setelah mutasi; mutation+audit belum atomik dan kegagalan audit dapat meninggalkan mutasi tanpa audit. Belum di produksi.
- **Task 2.5 selesai (13 Nov 2026, commit `6498211`).** Beranda kini memuat konten terbaru; route statis `/pengumuman/`, `/pengumuman/baca/?id=`, `/materi/`, dan `/materi/baca/?id=` menyediakan daftar/detail runtime tanpa dynamic segment. Klien API defensif memakai URL relatif, pagination/filter topik, AbortSignal, dan pesan galat tetap. Perender teks biasa memecah paragraf, mempertahankan HTML berbahaya sebagai teks ter-escape, hanya menautkan HTTP(S), dan tidak memakai `dangerouslySetInnerHTML`. Tersedia loading, kosong, galat+retry, invalid-ID, pagination, WIB, dan kontrol berlabel. Remote `npm ci`, 43 tes, lint, typecheck, build bersih dua kali, lima output statis, dan `npm audit --omit=dev` (0) hijau. **Batas:** tidak ada paket browser automation, jadi viewport/network browser nyata belum diuji; bukti terkuat adalah build statis dua kali, assertions output, dan tes Node. RED awal dari subagent yang timeout tidak dapat diverifikasi dan tidak diklaim. Belum di produksi.
- **Task 2.6 selesai (14 Nov 2026).** Klien `frontend/src/lib/api-client/admin-content.ts` (34 uji: isian kosong dari server diisi bawaan, `updatedAt` dikirim balik apa adanya, CSRF di setiap tulis, galat jadi teks tetap, validasi sama dengan server termasuk hitungan karakter emoji). UI: dasbor `/admin/` berisi daftar Pengumuman dan Materi (status teks+ikon+warna, Terbitkan/Tarik ke draf, Hapus dengan konfirmasi di baris, paginasi), formulir `/admin/pengumuman/ubah/` dan `/admin/materi/ubah/` (`?id=` untuk ubah; penghitung karakter; Umum = tanpa topik; konflik 409 menawarkan muat data terbaru; sesi habis dialihkan ke login). Label publik materi tanpa topik diganti 'Tanpa topik' menjadi 'Umum' sesuai keputusan. E2E di `cbt-vps` dengan klien asli, API Go asli, dan database uji lulus. **Belum:** cek tampilan di browser nyata (360px, zoom 200%, keyboard) — dilakukan saat deploy Task 2.7.
- **Task 2.9 selesai (14 Nov 2026).** RED: 6 uji skema lampiran gagal (tabel belum ada). GREEN di `cbt-vps` dengan database uji: migrasi `00007_create_material_attachments` — `storage_key` UUID unik (nama di disk dibuat server, bukan dari pengguna), `kind` hanya pdf/ppt/pptx dan harus cocok dengan `content_type` (makro `pptm` ditolak), ukuran 1 B s.d. 25 MiB, SHA-256 32 byte, nama asli 1–200 karakter tanpa `/`, `\`, atau karakter kontrol; maksimal 5 lampiran per materi lewat trigger yang mengunci baris materi lebih dulu; hapus materi ikut menghapus baris lampiran (kunci penyimpanan masih bisa dibaca saat hapus untuk membuang berkas). Uji konkurensi 12 sisipan serentak menyimpan tepat 5, lulus 3 kali; **uji mutan** (kunci `FOR UPDATE` dibuang) membuat uji itu gagal 4 dari 5 kali. `down-to 6` bersih tanpa menyentuh data 2A. Seluruh suite, vet, build hijau.
  - Direktori data dibuat di VPS: `/home/ubuntu/informatika-monitoring-data/attachments` (0750, `ubuntu:ubuntu`), sisa disk 23,7 GB. Belum dipakai aplikasi.
  - **Temuan untuk Task 2.13:** unit systemd memakai `ProtectSystem=strict` dan `ProtectHome=read-only`, jadi layanan **tidak bisa menulis** ke direktori data. Unit harus ditambah `ReadWritePaths=/home/ubuntu/informatika-monitoring-data` (perubahan konfigurasi server, butuh izin), dan `deploy.sh` harus memeriksa direktori ini.
  - Migrasi 7 belum di produksi; ikut deploy 2B.
- **Task 2.10 selesai (14 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps`: paket `backend/internal/attachments` lulus, juga `-race`. `Detect` menentukan jenis dari isi (PDF `%PDF-1.`/`%PDF-2.` di awal; PPTX: ZIP dengan `[Content_Types].xml` bertipe presentasi dan `ppt/presentation.xml`; PPT: Compound File yang direktorinya dibaca sungguhan dan wajib punya aliran `PowerPoint Document`) dan ekstensi wajib cocok; makro ditolak (`vbaProject.bin`, tipe `macroEnabled`, penyimpanan `_VBA_PROJECT_CUR`/`VBA`/`Macros`); `.pptm`/`.ppsx`/tanpa ekstensi/`.pdf.exe` ditolak. Batas pengaman: tak ada isi ZIP yang dibuka kecuali satu XML kecil (bom ZIP 50 MB tidak dibuka), XML maks 1 MiB, maks 20.000 entri, rantai sektor OLE dicek terhadap putaran dan sektor di luar berkas. `Store.Save` mengalirkan unggahan ke berkas sementara di folder yang sama (tak pernah utuh di memori), menghitung SHA-256, menolak >25 MiB, kuota 2 GiB, dan sisa disk < 1 GiB + ukuran maksimal, lalu `rename` atomik ke nama UUIDv4 acak tanpa ekstensi (mode 0640); gagal di titik mana pun tidak meninggalkan berkas. `Open`/`Remove` menolak apa pun selain UUID kanonis. `NewStore` menolak folder yang tidak ada atau terbuka untuk pengguna lain. `CleanName` membuang jalur dan karakter kontrol, maks 200 karakter dengan ekstensi dipertahankan.
  - Berkas uji: sintetis untuk PDF/PPT/PPTX, plus **PPTX asli** `testdata/real-python-pptx.pptx` dibuat dengan python-pptx 1.0.2 dari template PowerPoint bawaan. **Belum diuji dengan berkas .ppt asli** (tidak ada pembuatnya di lingkungan ini); dilakukan di cek browser Task 2.15 dengan berkas user.
  - Uji mutan: cek makro PPTX dibuang, dua uji gagal.
  - `golang.org/x/sys` kini dependensi langsung (versi sama, v0.50.0) untuk `statfs`.
  - Batas: tidak ada pemindai virus; PDF tidak diperiksa isinya (JavaScript di PDF tidak ditolak), aman karena diunduh sebagai lampiran dengan `nosniff` (Task 2.12).
- **Task 2.11 selesai (14 Nov 2026).** RED: `attachments.Service`, `NewService`, `ErrMaterialNotFound`/`ErrLimit`/`ErrBusy`, dan `Deps.Attachments` belum ada (build gagal). GREEN di `cbt-vps` dengan database uji nyata: 8 uji layanan + 5 uji HTTP + 1 uji config; seluruh suite lulus 2 kali, `-race` paket `attachments` dan `http`, vet, build `api`/`adminctl` hijau.
  - Endpoint: `GET`/`POST /api/v1/admin/materials/{id}/attachments` (multipart, satu bagian `file`) dan `DELETE /api/v1/admin/attachments/{id}`. Semua di belakang `Require` (sesi; POST/DELETE wajib CSRF + Origin). Respons tidak pernah memuat kunci penyimpanan.
  - Terbukti: anonim 401, tanpa CSRF atau Origin asing 403; 201 + `Location`; 25 MiB + 1 byte 413 (dialirkan dari pembangkit, tidak pernah utuh di memori); isi palsu dan `.pdf.exe` 415 `UNSUPPORTED_FILE_TYPE`; JSON 415 `UNSUPPORTED_MEDIA_TYPE`; isian salah 400; materi tak ada 404; lampiran ke-6 409 `ATTACHMENT_LIMIT`; kuota penuh 507 `STORAGE_QUOTA_EXCEEDED`; unggahan kedua saat satu berjalan `ErrBusy` (uji layanan; pemetaan ke HTTP 503 + `Retry-After` belum diuji di HTTP). Penolakan tidak meninggalkan berkas maupun baris audit.
  - Urutan tulis: berkas dulu lalu baris; bila sisip gagal berkas dihapus. Hapus lampiran: baris dulu lalu berkas. Hapus materi (bila lampiran aktif) mengunci baris materi, membaca kunci berkas, menghapus materi, commit, lalu menghapus berkasnya. Audit `content.attachment_uploaded`/`content.attachment_deleted` dengan detail `entityId`, `materialId`, `title` (nama berkas).
  - **Uji mutan:** pembersihan setelah sisip gagal dibuang → 1 uji gagal; penghapusan berkas saat hapus materi dibuang → 2 uji gagal (layanan dan HTTP).
  - Tenggat baca/tulis 10 menit dipasang per rute lewat `http.ResponseController`; **belum teruji di server nyata** (perekam uji tidak mendukungnya). Diuji saat deploy (Task 2.15) bersama batas Caddy (Task 2.13).
  - Konfigurasi baru `ATTACHMENTS_DIR` (kosong = rute lampiran nonaktif; direktori harus ada dan tertutup untuk pengguna lain, kalau tidak API berhenti saat mulai). `deploy.sh` belum menerima kunci ini di `.env` — ditambahkan di Task 2.13 bersama `ReadWritePaths` systemd. Jadi deploy sebelum 2.13 aman: lampiran tetap mati.
  - OpenAPI 0.4.0 memuat dua path baru dan skema `Attachment` (YAML valid, semua `$ref` terselesaikan).
  - Batas diketahui: crash di antara tulis berkas dan sisip baris bisa meninggalkan berkas yatim (tidak pernah baris tanpa berkas); pembersih berkala belum ada. Audit tetap fail-open seperti 2A.
- **Task 2.12 selesai (14 Nov 2026).** RED perilaku: unduh dari materi terbit memberi 404 karena rute belum ada (uji berkas-hilang lolos semu saat RED; kini lolos karena alasan yang benar). GREEN di `cbt-vps` dengan database uji nyata: 3 uji HTTP baru; seluruh suite lulus 2 kali, `-race` paket `attachments`/`http`, vet, build `api`/`adminctl`, gofmt hijau.
  - Endpoint publik: `GET /api/v1/public/materials/{id}/attachments` (hanya `id`, `name`, `kind`, `size`) dan `GET /api/v1/public/attachments/{id}`. Keduanya hanya untuk materi terbit; draf, ditarik ke draf, ID tidak ada, ID cacat, dan berkas hilang dari disk semuanya JSON 404 (berkas hilang dicatat di log).
  - Header unduhan terbukti: `Content-Type` dari hasil validasi, `Content-Disposition: attachment` dengan nama ASCII cadangan + `filename*` UTF-8 (tanda kutip dan huruf non-ASCII aman), `nosniff`, `Content-Security-Policy: default-src 'none'; sandbox`, `Cache-Control: no-cache` (tarik ke draf langsung berlaku), `ETag` = SHA-256. Range 206 dengan `Content-Range` benar, range di luar ukuran 416, `If-None-Match` 304, `If-Range` basi memberi 200 penuh.
  - **Uji mutan:** filter `status = 'published'` di query unduh dibuang → uji draf gagal (draf terunduh 200).
  - Tenggat tulis 10 menit per rute; seperti 2.11, belum teruji di server nyata (dicek di Task 2.15).
  - OpenAPI 0.4.0: dua path publik dan skema `PublicAttachment` (YAML valid, semua `$ref` terselesaikan).
- **Task 2.13 disiapkan (14 Nov 2026), produksi tidak disentuh.** Aktivasi butuh izin user (langkah di `docs/DEPLOYMENT.md`, bagian *Material attachments*).
  - `deploy/Caddyfile`: `request_body` 27 MB hanya untuk `POST /api/v1/admin/materials/<angka>/attachments`, 1 MB untuk semua lainnya. `caddy validate` dan `caddy fmt` bersih. **Diuji nyata** dengan Caddy 2.11.9 di port loopback lain (admin API mati) dan upstream palsu: badan kecil 200; 2 MB ke login 413; 2 MB dan 25 MiB + 64 KiB ke jalur unggah diteruskan utuh; 28 MB ke jalur unggah 413; PUT, jalur mirip (`/attachments/x`), dan id bukan angka 413.
  - `deploy/informatika-monitoring-api.service`: `ReadWritePaths=/home/ubuntu/informatika-monitoring-data/attachments` (satu-satunya jalur tulis).
  - `deploy/deploy.sh`: `.env` menerima `ATTACHMENTS_DIR` (opsional, sekali, jalur absolut polos). Bila diisi, pre-flight menolak direktori di dalam checkout, symlink, milik user lain, atau terbuka untuk user lain, dan menolak bila `ReadWritePaths` unit yang berjalan tidak mencakupnya. Sebelum mutasi, semua lampiran (hanya nama UUID; berkas `.upload-*` dilewati) di-snapshot ke `backups/pre-<rilis>.attachments/` dengan hard link (tanpa tambahan disk; disalin bila beda filesystem), dicek nama+ukuran per berkas, lalu dipangkas ke 10 terbaru. `--check` merehearse snapshot ke direktori sementara. Pesan rollback menyebut snapshot.
  - Uji fungsi deploy: **89 lulus** (sebelumnya 59): parsing `ATTACHMENTS_DIR`, `unit_allows_write`, `check_attachments_dir`, snapshot + **pemulihan** (berkas dihapus lalu dipulihkan sesuai runbook, isi dan mode 0640 sama), snapshot tidak menimpa yang ada, pemangkasan. Tiga uji mutan (cek unit dibuang, filter UUID dibuang, cek izin user lain dibuang) masing-masing menggagalkan satu uji yang tepat. `bash -n` bersih.
  - **Belum:** pemasangan unit/Caddyfile dan baris `.env` di produksi; uji unggah 25 MB lewat Caddy produksi ke app sungguhan; snapshot di produksi (masih 0 berkas).
  - **Temuan:** proses uji lama `/tmp/informatika-monitoring-task06` (dari Task 0.6, user `ubuntu`) masih mendengarkan di `*:18300` sejak sekitar 12 Nov. Bukan produksi; belum dimatikan, menunggu persetujuan user.
- **Task 2.13 diaktifkan di produksi (14 Nov 2026, atas izin user).** Urutan: `git pull` ke `57438e0`; unit lama dicadangkan ke `/etc/systemd/system/informatika-monitoring-api.service.bak-20261114`, unit baru dipasang, `daemon-reload`, `ReadWritePaths` terbaca benar; Caddyfile lama dicadangkan ke `/etc/caddy/Caddyfile.bak-20261114`, yang baru lolos `caddy validate`, `reload` (situs tetap 200); satu baris `ATTACHMENTS_DIR` ditambahkan ke `.env` (tetap 0600); `deploy.sh --remote` exit 0.
  - Rilis `20261114T030336Z-57438e043577`, **migrasi versi 7**. Cadangan `backups/pre-20261114T030336Z-57438e043577.dump` (20651 byte) dan snapshot lampiran `...attachments` (0 berkas, 0700) ditulis sebelum mutasi. Pre-flight mengonfirmasi direktori lampiran dapat ditulis layanan. Log deploy tidak memuat pola DSN/sandi lalu dihapus.
  - Dari luar lewat Caddy: unduh ID tak dikenal → `Lampiran tidak ditemukan.` (rute aktif); 2 MB anonim ke jalur unggah diteruskan ke app → 401; 2 MB ke login → 413; Beranda, `/materi/`, `/admin/login/` 200.
  - **Belum:** unggah sungguhan dengan sesi admin (berkas tertulis di disk produksi) dan unduh; itu bagian Task 2.15 bersama UI Task 2.14.
- **Task 2.14 selesai (14 Nov 2026).** Tes klien ditulis bersamaan dengan klien, jadi **tidak ada RED ketat** yang diklaim. GREEN di `cbt-vps`: `npm ci`, **90 uji** Node (13 baru untuk lampiran), lint, typecheck, build statis hijau.
  - Klien `frontend/src/lib/api-client/attachments.ts`: cek awal nama (.pdf/.ppt/.pptx, huruf besar/kecil bebas; .pptm/.ppsx/.pdf.exe ditolak), berkas kosong, dan >25 MiB sebelum mengirim apa pun; unggah multipart lewat `XMLHttpRequest` agar ada progres dan bisa dibatalkan (fetch tidak bisa melaporkan progres unggah); setiap kode galat server (`ATTACHMENT_LIMIT`, `MACRO_NOT_ALLOWED`, `STORAGE_QUOTA_EXCEEDED`, `SERVICE_BUSY`, dll.) jadi teks tetap; badan non-JSON seperti 413 dari Caddy dipetakan dari status; respons sukses divalidasi.
  - Admin: bagian **Lampiran** di `/admin/materi/ubah/?id=` (hanya materi yang sudah disimpan; materi baru diberi petunjuk). Daftar dengan jenis, ukuran, waktu WIB; hitungan "n dari 5"; pilih berkas, tombol unggah, bilah `<progress>` berlabel persen + ukuran, **Batalkan unggahan**; hapus dengan konfirmasi di baris; formulir unggah diganti pesan saat sudah 5; galat fokus ke pesan, sesi habis dialihkan ke login.
  - Publik: bagian **Lampiran** di `/materi/baca/?id=` berisi tautan unduh (`download`) dengan label jenis dan ukuran; tidak tampil bila kosong; galat + coba lagi.
  - **E2E nyata di `cbt-vps`** (biner `api`/`adminctl` asli, database uji, direktori lampiran sementara, klien frontend asli lewat Node): login; unggah **PPTX asli** (28.356 B) dan PDF; PDF palsu ditolak dengan pesan tetap; daftar admin dan publik benar; unduhan PPTX identik byte-per-byte dengan `filename*` UTF-8; hapus → sisa 1, hapus ulang `not_found`; berkas di disk 1; halaman `/materi/baca/` dan `/admin/materi/ubah/` 200. Database uji dan direktori dibersihkan sesudahnya.
  - **Belum:** tampilan di browser nyata (progres, 360px, keyboard) dan unggahan lewat Caddy produksi — Task 2.15.
