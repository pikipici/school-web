# Fase 3 — Modul Progres Tugas Kelompok (Rencana)

> Status: **DISETUJUI 14 Nov 2026** (user mengikuti semua rekomendasi). **3A ditutup 15 Nov 2026** dengan pengecualian (`docs/F3A_ACCEPTANCE.md`); **3B ditutup 18 Nov 2026** dengan pengecualian (`docs/F3B_ACCEPTANCE.md`); 3C berikutnya.
> Sumber: `PLAN.md` bagian 1–7, 9, 10, 14 (AC01–AC24); `DESIGN.md`; pola Fase 1–2 (sesi, CSRF, audit, `?id=` untuk ekspor statis).

## Tujuan

Guru mencatat kelas, kelompok, pengumpulan PPT/PDF, buku tulis, dan hasil pemeriksaan dari dashboard. Publik melihat tabel ringkas di `/progres/`. Tiap kelompok membaca catatan revisi miliknya sendiri lewat kode akses. Semua status tetap diisi manual oleh guru; tidak ada Gmail dan tidak ada unggah dari siswa.

## Pembagian

`PLAN.md` bagian 13 memecah modul ini menjadi lima fase lama (2–6). Saya usulkan empat bagian, masing-masing dideploy dan ditutup sendiri:

| Bagian | Isi | Fase lama `PLAN.md` | Risiko utama |
|---|---|---|---|
| **3A** | Tahun ajaran, kelas, kelompok; tabel publik `/progres/` | 2 | Data privat (email kelompok) bocor ke publik |
| **3B** | Pencatatan pengumpulan PPT/PDF per putaran + buku tulis | 3 | Status salah hitung; revisi menghapus bukti lama |
| **3C** | Pemeriksaan: status review, temuan draf/terbit | 4 | Draf terlihat siswa; transisi status tidak sah |
| **3D** | Portal kelompok dengan kode akses | 5 | Kelompok A membaca temuan kelompok B |

Di luar Fase 3 (usul jadi Fase 4): sesi presentasi + pertanyaan antarkelompok, rekap + ekspor CSV, halaman audit admin.

## Keputusan yang saya usulkan

| # | Hal | Usulan | Alasan |
|---|---|---|---|
| K1 | Tahun ajaran | Seed `2026/2027` aktif; guru bisa menambah; tepat satu aktif | AC20; publik hanya melihat tahun aktif |
| K2 | Kelas | Kode bebas (mis. `9A`), unik per tahun ajaran | AC20 |
| K3 | Kelompok | Nama (mis. `Kelompok 1`), unik per kelas; topik K1–K7 wajib; email kelompok opsional | Topik menentukan isi tugas |
| K4 | Email kelompok | Disimpan untuk guru saja; form menyarankan format `[nama].[kelas].2627.Informatika@gmail.com` | AC04: tidak pernah di respons publik/portal |
| K5 | Data siswa | **Tidak menyimpan nama anggota**; peran anggota (`PLAN.md` 2.5) ditunda | Data siswa paling sedikit; tidak dibutuhkan untuk status |
| K6 | Hapus kelompok/kelas | Hapus hanya bila belum ada catatan apa pun; selain itu **nonaktifkan** (hilang dari publik, riwayat tetap) | `PLAN.md` 9 aturan 4 |
| K7 | Status pengumpulan | **Dihitung** dari semua putaran: BELUM / SEBAGIAN / LENGKAP; tidak disimpan | Aturan 9.1: revisi PPT tidak menghapus PDF lama (AC08) |
| K8 | Putaran | 0 = awal, 1+ = revisi; tiap putaran centang PPT dan/atau PDF, waktu terima (WIB di form), pengirim, subjek, nama berkas, catatan internal | `PLAN.md` 6B/6D |
| K9 | Koreksi putaran | Putaran bisa diubah; dihapus hanya putaran terakhir dan hanya bila belum dirujuk temuan; semua tercatat audit | Salah ketik perlu bisa diperbaiki |
| K10 | Status review | Lima status `PLAN.md` 5.2; aturan: `PERLU_REVISI` butuh ≥1 temuan wajib terbit dan terbuka; `DISETUJUI` butuh LENGKAP dan 0 temuan wajib terbuka | AC09; ditegakkan di Go dan diuji |
| K11 | Temuan | Jenis PPT/PDF/Buku/Presentasi; lokasi wajib; masalah wajib; tindakan wajib; prioritas **Wajib/Saran**; status Terbuka/Selesai | `PLAN.md` 5.2, 6C |
| K12 | Terbit temuan | Per kelompok: tombol **Terbitkan evaluasi** menerbitkan semua draf sekaligus, dengan konfirmasi; temuan terbit tidak dihapus, hanya **diarsipkan** | Satu langkah sengaja; aturan 9.4 |
| K13 | TUNTAS | Dihitung: LENGKAP + DISETUJUI + buku diterima | AC10; tidak bisa diset tangan |
| K14 | Tabel publik | Kolom `PLAN.md` 5.4; filter kelas dan status pengumpulan; cari nama kelompok; tanpa email, temuan, atau catatan | AC04 |
| K15 | Kode akses | 12 karakter acak dari alfabet tanpa huruf mirip (mis. `K7QM-4XWD-9TRA`), ±60 bit; disimpan sebagai HMAC-SHA256 dengan kunci server; **ditampilkan sekali** saat dibuat/rotasi | Cukup kuat dengan rate limit; tidak bisa dibaca ulang dari DB |
| K16 | Sesi kelompok | Cookie terpisah `__Host-group_session`, HttpOnly, SameSite=Strict, maks **7 hari**; rotasi kode mencabut semua sesi kelompok itu; portal **baca saja** | Siswa membuka sesekali; tidak bisa mengubah apa pun |
| K17 | Batas percobaan kode | Pakai paket `ratelimit` Fase 1: per IP dan global; kunci bertahap | AC13; kode tidak terikat nama akun |
| K18 | Alamat halaman | Publik `/progres/`; portal `/progres/masuk/` dan `/progres/kelompok/`; admin `/admin/kelas/`, `/admin/kelompok/`, `/admin/kelompok/detail/?id=` | Ekspor statis tanpa segmen dinamis (seperti Fase 2) |
| K19 | ID | `bigint`, seperti Fase 1–2 | Konsisten |
| K20 | Audit | Setiap mutasi + rotasi kode + masuk/keluar kelompok; detail tanpa kode akses, email, atau isi temuan | AC11 |

## Temuan teknis

1. **`/progres/` saat ini statis** (tiga jalur + `ProjectSummary` dari `GET /api/v1/public/summary` yang selalu nol). 3A mengganti isinya dengan data nyata; endpoint ringkasan diisi angka sebenarnya.
2. **Kunci HMAC kode akses** butuh rahasia baru di `.env` (mis. `GROUP_CODE_KEY`, 32 byte acak). `deploy.sh` harus menerima dan memvalidasi kunci ini — perubahan `.env` produksi, butuh izin (3D).
3. **Transaksi**: catat putaran + audit, terbitkan evaluasi + ubah status, rotasi kode + cabut sesi harus satu transaksi. Audit Fase 2 masih fail-open di luar transaksi; untuk Fase 3 saya usulkan audit ditulis **di dalam transaksi yang sama** untuk aksi-aksi ini.
4. **Klik ganda**: catat putaran dan terbitkan evaluasi rawan terkirim dua kali. Usul: putaran memakai nomor yang dikirim klien (`roundNumber` berikutnya) dengan constraint unik, sehingga kiriman kedua mendapat 409.
5. **Waktu terima** diketik guru dalam WIB; disimpan UTC; tidak boleh di masa depan lebih dari beberapa menit.
6. **Cache publik**: tabel publik `no-store` atau cache sangat singkat; portal dan admin selalu `no-store`.
7. **CSP** tetap `enforce`; halaman baru mengikuti pola hash otomatis.

## Endpoint (garis besar)

| Method | Path | Siapa |
|---|---|---|
| GET | `/api/v1/public/progress` | publik: tahun aktif, kelas, baris kelompok ringkas (`?class=&status=&q=`) |
| GET | `/api/v1/public/summary` | publik: angka nyata (total, belum, sebagian, lengkap, tuntas) |
| GET/POST | `/api/v1/admin/academic-years` | admin |
| PATCH | `/api/v1/admin/academic-years/{id}` | admin (aktifkan) |
| GET/POST | `/api/v1/admin/classes` | admin |
| GET/PATCH/DELETE | `/api/v1/admin/classes/{id}` | admin |
| GET/POST | `/api/v1/admin/groups` | admin (`?class=`) |
| GET/PATCH/DELETE | `/api/v1/admin/groups/{id}` | admin |
| POST | `/api/v1/admin/groups/{id}/access-code` | admin: buat/rotasi, kembalikan kode **sekali** |
| GET/POST | `/api/v1/admin/groups/{id}/submissions` | admin (3B) |
| PATCH/DELETE | `/api/v1/admin/submissions/{id}` | admin (3B) |
| PUT | `/api/v1/admin/groups/{id}/notebook` | admin (3B) |
| GET/POST | `/api/v1/admin/groups/{id}/findings` | admin (3C) |
| PATCH | `/api/v1/admin/findings/{id}` | admin (3C: ubah, selesai, arsip) |
| POST | `/api/v1/admin/groups/{id}/publish` | admin (3C: terbitkan semua draf) |
| PATCH | `/api/v1/admin/groups/{id}/review-status` | admin (3C) |
| POST | `/api/v1/group/access` | publik: tukar kode dengan sesi (3D) |
| POST | `/api/v1/group/logout` | kelompok (3D) |
| GET | `/api/v1/group/me` | kelompok: status + temuan **terbit** + riwayat putaran tanpa catatan internal (3D) |

Mutasi admin: sesi + CSRF + Origin + audit. `PATCH` memakai `updatedAt` dan 409 saat konflik, seperti Fase 2.

## Task

### 3A — Kelas, kelompok, tabel publik

| Task | Isi | Uji |
|---|---|---|
| 3.1 **DONE** | Migrasi `academic_years` (+ seed 2026/2027), `classes`, `groups` | uji skema: unik, satu tahun aktif, topik wajib, down/up |
| 3.2 **DONE** | Paket `progress`: validasi kelas/kelompok, nonaktif vs hapus, query publik aman | TDD + PostgreSQL nyata |
| 3.3 **DONE** | API admin kelas/kelompok + API publik `progress`/`summary` + audit | TDD HTTP: 401/403/409/422; publik tanpa email (uji kunci JSON) |
| 3.4 **DONE** | UI admin kelas/kelompok; `/progres/` memuat tabel, filter, cari, keadaan kosong/galat | klien API diuji; lint/typecheck/build |
| 3.5 **DONE** | Deploy + cek browser + `docs/F3A_ACCEPTANCE.md` (AC01, AC04 sebagian, AC18, AC20) | semua gerbang |

### 3B — Pengumpulan dan buku tulis

| Task | Isi | Uji |
|---|---|---|
| 3.6 **DONE** | Migrasi `submission_rounds`, `group_notebooks` | uji skema: unik `(group, round)`, minimal satu berkas per putaran |
| 3.7 **DONE** | Status pengumpulan terhitung; catat/ubah/hapus putaran; buku tulis; audit dalam transaksi | TDD: AC02, AC03, AC08; klik ganda 409 |
| 3.8 **DONE** | UI admin "Catat Pengumpulan" + buku tulis di detail kelompok; publik menampilkan status | lint/typecheck/build + E2E |
| 3.9 **DONE** | Deploy + cek browser + `docs/F3B_ACCEPTANCE.md` | semua gerbang |

### 3C — Kesalahan tugas (temuan), tampil publik

> **Disederhanakan 18 Nov 2026** atas arahan user ("gausa terlalu advance, gua cuma perlu website buat ngasi tau siswa letak kesalahan tugas mereka"). Fase 4 (tanya jawab, rekap, CSV, halaman audit) **dibatalkan**; lihat `PHASE4_PLAN.md`.
>
> **Kode akses dibuang 18 Nov 2026** (user: "Buang kode akses, semua kesalahan tampil publik"). 3D dibatalkan. Temuan yang sudah **terbit** bisa dibaca siapa pun di halaman publik kelompok. Yang tetap tersembunyi: temuan **draf**, email kelompok, catatan internal guru, pengirim/subjek/nama berkas putaran. K15–K17 dan AC07, AC13 tidak berlaku.

| Task | Isi | Uji |
|---|---|---|
| 3.10 | Migrasi `evaluation_findings` + status pemeriksaan di kelompok; paket + API admin: tambah/ubah/hapus temuan draf, terbitkan semua draf, tandai selesai, ubah status pemeriksaan; TUNTAS terhitung; audit dalam transaksi. API publik `GET /api/v1/public/groups/{id}`: nama, kelas, topik, tiga lajur, temuan **terbit** saja; baris tabel publik dapat `id` | TDD: AC05 (draf tidak pernah publik, uji mutan), AC06, AC09, AC10; kunci JSON publik persis tanpa email/catatan |
| 3.11 | Admin: bagian **Temuan** di halaman detail kelompok (jenis, lokasi, masalah, perbaikan, Wajib/Saran), **Terbitkan** dengan konfirmasi, **Selesai**, pilih status pemeriksaan. Publik: nama kelompok di `/progres/` jadi tautan ke `/progres/kelompok/?id=` berisi daftar kesalahan | uji klien; lint/typecheck/build; E2E; 360px |
| 3.12 | Deploy + cek browser + `docs/F3C_ACCEPTANCE.md` | semua gerbang |
| 3.13 | Panduan singkat guru (`docs/ADMIN_GUIDE.md`: buat kelas/kelompok, catat kiriman, tulis dan terbitkan kesalahan); satu uji alur nyata dari awal sampai siswa membaca kesalahannya | user membaca panduan; alur nyata lulus |

Temuan yang sudah terbit tidak dihapus; guru menandainya **Selesai** setelah siswa memperbaiki. Tidak ada fitur arsip terpisah.

Setiap task: uji gagal dulu, implementasi, uji/vet/build di `cbt-vps`, commit, catat di dokumen ini dan `PROGRESS.md`.

## Risiko dan mitigasi

1. **Email/temuan bocor ke publik.** DTO publik terpisah; uji membandingkan daftar kunci JSON persis.
2. **Kelompok membaca data kelompok lain.** `group/me` tidak menerima ID dari klien; identitas hanya dari sesi; uji AC07.
3. **Tebak kode akses.** ±60 bit + rate limit per IP dan global + audit percobaan gagal.
4. **Kode tersebar.** Rotasi satu klik mencabut semua sesi; kode lama langsung mati.
5. **Status salah hitung.** Status pengumpulan dan TUNTAS dihitung, bukan disimpan; uji kombinasi putaran.
6. **Draf terlihat siswa.** Query portal memfilter `published`; uji mutan seperti 2B.
7. **Salah input guru.** Putaran dan temuan bisa dikoreksi; semuanya beraudit; temuan terbit diarsipkan, bukan dihapus.
8. **Data siswa.** Tidak menyimpan nama anggota; email kelompok hanya admin.

## Keputusan user (14 Nov 2026)

User mengikuti semua rekomendasi:

1. Dibagi 3A–3D; partisipasi tanya jawab, rekap, dan CSV menjadi Fase 4.
2. Nama anggota kelompok **tidak** disimpan (K5).
3. Sesi kelompok maksimal **7 hari** (K16).
4. Jumlah kelas/kelompok belum diberikan; uji tampilan memakai data contoh **4 kelas × 7 kelompok** (28 baris) di database uji, bukan data siswa.
5. Kelompok **bisa melihat riwayat putarannya sendiri** (tanggal terima WIB, PPT/PDF), tanpa pengirim, subjek, nama berkas, atau catatan internal.

Status rencana: **DISETUJUI 14 Nov 2026.**

## Catatan pelaksanaan

- **Task 3.1 selesai (14 Nov 2026).** RED: 5 uji skema baru gagal (`relation "academic_years" does not exist`). GREEN di `cbt-vps` dengan database uji: migrasi `00008_create_progress_groups`.
  - `academic_years`: label `YYYY/YYYY` dengan tahun kedua = pertama + 1, unik; default tidak aktif; **paling banyak satu aktif** (indeks unik parsial; "minimal satu" dijaga di Go); seed `2026/2027` aktif; tukar tahun aktif dalam satu `UPDATE` boleh.
  - `classes`: kode 1–20 karakter (huruf/angka/spasi/titik/strip, diawali huruf/angka), unik per tahun **tanpa beda huruf besar/kecil**; nama opsional ≤ 80 tanpa karakter kontrol; tahun yang punya kelas tidak bisa dihapus.
  - `study_groups` (bukan `groups`, karena `GROUPS` kata kunci SQL): nama 1–60 karakter tanpa karakter kontrol, unik per kelas tanpa beda huruf; topik **wajib**; email opsional, huruf kecil, format email, ≤ 254; urutan ≥ 0; default aktif; kelas yang punya kelompok dan topik yang dipakai tidak bisa dihapus. Tidak ada kolom nama anggota.
  - `down-to 7` menghapus ketiga tabel tanpa menyentuh Materi; `up` ulang tidak menggandakan seed. `goose validate`, `sqlc generate`, seluruh suite 2 kali, vet, build hijau.
  - **Uji mutan:** indeks satu-aktif dibuang → uji tahun ajaran gagal; indeks nama tanpa `lower()` → uji kelompok gagal.
  - Kolom kode akses ditambahkan nanti di Task 3.14. Migrasi 8 belum di produksi; ikut deploy Task 3.5.
- **Task 3.2 selesai (15 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps` dengan database uji: paket `backend/internal/progress` + `sql/queries/progress.sql`; 6 uji lulus, seluruh suite 2 kali, `-race`, vet, build, gofmt hijau.
  - Validasi di Go (sama dengan DB): tahun `YYYY/YYYY` berurutan; kode kelas 1–20; nama kelas ≤ 80; nama kelompok 1–60 tanpa karakter kontrol; email dirapikan ke huruf kecil, kosong = tidak ada; topik dan kelas wajib; urutan ≥ 0. Semua dipangkas spasinya.
  - Tahun ajaran: daftar (terbaru dulu), tambah (tidak aktif), **aktifkan** dalam satu transaksi (matikan yang lama dulu, baru nyalakan yang baru; ID tak ada → `ErrNotFound` dan tahun aktif tidak berubah).
  - Kelas/kelompok: buat (selalu aktif), ubah dengan `updatedAt` (`ErrConflict` vs `ErrNotFound`), nonaktifkan lewat ubah, hapus (`ErrInUse` bila kelas masih punya kelompok; kelompok nanti juga `ErrInUse` saat sudah punya catatan 3B/3C). Nama ganda tanpa beda huruf → `ErrDuplicate`; tahun/kelas/topik tak dikenal → `ErrInvalidReference`. Kelompok bisa dipindah kelas.
  - Publik: hanya tahun aktif, kelas aktif, kelompok aktif; urut kelas lalu urutan kelompok; filter kode kelas (tanpa beda huruf) dan cari nama (teks biasa, `%`/`_` tidak jadi wildcard); maks 60 karakter. Baris publik **persis** `classCode, complete, name, notebook, review, submission, topic, updatedAt` — tanpa ID, email, atau catatan (diuji lewat JSON). Di 3A semua kelompok `BELUM` / `BELUM_DICEK` / `BELUM_DITERIMA`. Ringkasan menghitung total, belum, sebagian, lengkap, tuntas. Daftar kosong dikirim sebagai `[]`.
  - **Uji mutan:** filter kelompok aktif dibuang → gagal; filter tahun aktif dibuang → gagal; email disisipkan ke nama publik → gagal.
- **Task 3.3 selesai (15 Nov 2026).** RED: build gagal (`unknown field Progress in struct literal of type httpapi.Deps`). GREEN di `cbt-vps` dengan database uji: 3 uji HTTP baru; seluruh suite 2 kali, `-race` paket `http`/`progress`, vet, build, gofmt hijau.
  - Admin (sesi; POST/PATCH/DELETE wajib CSRF + Origin): `GET/POST /academic-years`, `POST /academic-years/{id}/activate` (204; menggantikan rencana `PATCH`), `GET/POST /classes` (`?year=`, bawaan tahun aktif), `GET/PATCH/DELETE /classes/{id}`, `GET/POST /groups` (`?class=` wajib), `GET/PATCH/DELETE /groups/{id}`. POST 201 + `Location`; PATCH wajib `active` dan `updatedAt`. Galat: 400 `INVALID_REQUEST`/`VALIDATION_ERROR` (query), 409 `CONFLICT`/`DUPLICATE`/`IN_USE`, 422 `VALIDATION_ERROR`/`INVALID_REFERENCE`, 404, 415, 413. Respons tanpa `omitempty` (email `null` bila kosong).
  - Audit: `progress.year_created`, `year_activated`, `class_created/updated/deleted`, `group_created/updated/deleted`, detail `entityId` + `title` (kode/nama/label, **tanpa email**); permintaan gagal tidak diaudit. Audit tetap fail-open seperti 2A (transaksi audit dijadwalkan untuk aksi 3B–3D).
  - Publik: `GET /api/v1/public/progress` (`class`, `q`, `status` = BELUM/SEBAGIAN/LENGKAP/TUNTAS; nilai salah 400) dengan `Cache-Control: no-cache`; kunci baris persis seperti 3.2, tidak ada kata "email" di respons. `GET /api/v1/public/summary` kini memakai data nyata saat database ada: kunci lama `totalGroups`, `manualUpdate`, `updatedAt` (waktu kelompok terakhir, `null` bila kosong) ditambah `none`, `partial`, `full`, `complete` — klien frontend lama tetap cocok.
  - **Uji mutan:** audit `group_updated` dibuang → gagal; filter status diabaikan → gagal.
  - OpenAPI 0.5.0: 26 jalur, skema `PublicProgress`, `PublicGroup`, `AcademicYear`, `Class(Write)`, `Group(Write)`, respons `ProgressConflict`; YAML valid, semua `$ref` terselesaikan (validator spesifikasi tidak terpasang di VPS saat ini).
- **Task 3.4 selesai (15 Nov 2026).** Tes klien ditulis bersama klien, jadi **tidak ada RED ketat** yang diklaim. GREEN di `cbt-vps`: `npm ci`, **102 uji** Node (12 baru untuk `progress.ts`), lint (0 masalah), typecheck, build statis (rute baru `/admin/kelas/`, `/admin/kelompok/`).
  - Klien `frontend/src/lib/api-client/progress.ts`: validasi sama dengan Go (tahun, kode kelas, nama kelompok per karakter, email huruf kecil), saran email juknis (`kelompok1.9a.2627.informatika@gmail.com`), semua kode galat server (`DUPLICATE`, `IN_USE`, `INVALID_REFERENCE`, `CONFLICT`, …) jadi teks tetap; respons divalidasi per baris.
  - Publik `/progres/`: peringatan pembaruan manual, satu kalimat ringkasan angka (bukan kartu KPI, sesuai `DESIGN.md`), filter kelas + pengumpulan + cari nama (disaring di browser dari satu muatan), tabel tujuh kolom `PLAN.md` 5.4; status selalu teks + ikon + warna; di ≤ 900 px tabel jadi baris rekaman berlabel (tanpa geser samping). Keadaan memuat, galat + coba lagi, kosong, dan "tidak ada untuk pilihan ini". `ProjectSummary.tsx` dihapus (tidak dipakai lagi).
  - Admin: menu admin (Dasbor / Kelas dan kelompok). `/admin/kelas/`: pilih tahun, jadikan aktif (konfirmasi), tambah tahun; daftar kelas dengan jumlah kelompok, ubah di baris, aktifkan/nonaktifkan, hapus hanya bila 0 kelompok (konfirmasi). `/admin/kelompok/?kelas=`: daftar, tambah (topik wajib, email opsional dengan tombol isi format juknis, urutan), ubah di baris, aktifkan/nonaktifkan, hapus (konfirmasi). Konflik dan "tidak ditemukan" memuat ulang daftar; sesi habis dialihkan ke login.
  - **E2E nyata di `cbt-vps`** (biner asli, database uji, klien frontend asli lewat Node): **4 kelas × 7 kelompok** dibuat lewat API; publik 28 baris semuanya `BELUM` dan **tanpa `email`/`@`**; nama ganda → `duplicate`; ubah basi → `conflict`; hapus kelas berisi → `in_use`; satu kelompok + kelas 9D dinonaktifkan → publik 20 baris, kelas 9A–9C; filter `9b` + "kelompok 7" → satu baris; ringkasan 20; halaman `/progres/`, `/admin/kelas/`, `/admin/kelompok/?kelas=1` 200. Database uji dibersihkan.
  - **Belum:** tampilan di browser nyata (360 px, zoom 200%, keyboard) — Task 3.5.
- **Task 3.5 selesai (15 Nov 2026).** Deploy `0700f32` (rilis `20261114T223049Z-0700f3215c1b`, migrasi 8 setelah cadangan), cek Edge menemukan meluap 360 px dan kata terpotong; diperbaiki `eff11c6`, deploy ulang (rilis `20261115T002336Z-eff11c66953c`), diukur ulang lulus. Data uji `UJI` dibuat dan dihapus lewat UI; audit tercatat. Bukti dan pengecualian di `docs/F3A_ACCEPTANCE.md`. **3A siap ditutup, menunggu persetujuan user.**
- **3A ditutup 15 Nov 2026** atas instruksi user ("Tutup Fase 3A"), dengan pengecualian di `docs/F3A_ACCEPTANCE.md`.
- **Task 3.6 selesai (15 Nov 2026).** RED: 7 uji skema baru gagal (`relation "submission_rounds" does not exist`). GREEN di `cbt-vps` dengan database uji: migrasi `00009_create_submissions`; `goose validate`, `sqlc generate`, seluruh suite 2 kali, vet, build hijau.
  - `submission_rounds`: satu baris per putaran (0 = awal, 1+ = revisi), waktu terima wajib, pengirim opsional (email huruf kecil ≤ 254), subjek ≤ 300 tanpa karakter kontrol, centang PPT/PDF dengan **minimal satu**, nama berkas hanya boleh diisi untuk jenis yang dicentang (≤ 200, tanpa `/`, `\`, karakter kontrol), catatan internal ≤ 2000, `recorded_by` → `admin_users` (`SET NULL` bila admin dihapus; catatan tetap). Unik `(group_id, round_number)`; trigger mewajibkan **nomor berikutnya tanpa lompat** (putaran pertama harus 0) setelah mengunci baris kelompok.
  - `group_notebooks`: satu baris per kelompok (kunci = `group_id`), `is_received` wajib sejalan dengan `received_at`, catatan ≤ 1000, `updated_by` `SET NULL`.
  - Kelompok yang sudah punya putaran atau baris buku **tidak bisa dihapus** (`RESTRICT` → `IN_USE` dari 3.2); kelompok kosong tetap bisa. `down-to 8` menghapus kedua tabel tanpa menyentuh kelompok; `up` ulang bersih.
  - Uji konkurensi: 10 sisipan putaran 0 serentak → tepat 1 berhasil, lulus 3 kali. **Jujur:** indeks unik saja sudah cukup membuat uji ini lulus, jadi kunci `FOR UPDATE` di trigger tidak dibuktikan oleh uji ini (yang ia lindungi: dua sisipan berbeda nomor yang sama-sama membaca maks lama).
  - **Uji mutan:** cek nomor berikutnya dimatikan → uji "gap in rounds" gagal; cek minimal satu berkas dibuang → uji "no file in round" gagal.
  - Migrasi 9 belum di produksi; ikut deploy Task 3.9.
- **Task 3.7 selesai (15 Nov 2026).** RED: build gagal (`undefined: progress.Actor`). GREEN di `cbt-vps` dengan database uji: 6 uji paket + 2 uji HTTP baru; seluruh suite 2 kali, `-race` `progress`/`http`, vet, build, gofmt hijau.
  - Paket `progress`: `RecordRound` (nomor berikutnya wajib; sama dua kali atau lompat → `ErrConflict`), `UpdateRound` (`updatedAt` → `ErrConflict`/`ErrNotFound`), `DeleteRound` (hanya putaran terakhir → `ErrNotLast`), `Rounds`, `SubmissionOf`, `Notebook`, `SetNotebook` (diterima tanpa tanggal = sekarang; tidak diterima menghapus tanggal). Validasi: minimal satu berkas, nama berkas hanya untuk jenis yang dicentang, waktu terima maks 5 menit di masa depan, email pengirim huruf kecil, subjek/catatan dibatasi.
  - **Audit dalam transaksi yang sama** (keputusan teknis #3): baris kelompok dikunci, perubahan + `progress_updated_at` + baris audit di satu transaksi lewat `audit.Write` baru (mengembalikan galat; `Record` lama tetap fail-open). Uji: trigger yang menggagalkan sisipan audit → catat putaran dan buku tulis **dibatalkan** (0 putaran, buku tidak berubah). Aksi: `progress.round_recorded/updated/deleted`, `progress.notebook_updated`, detail `entityId`, `title` (nama kelompok), `round`/`received` — tanpa email, subjek, atau catatan.
  - Publik: lajur Pengumpulan **dihitung dari semua putaran** (AC02 PPT → SEBAGIAN, AC03 + PDF → LENGKAP, AC08 revisi PPT saja tetap LENGKAP); lajur Buku Tulis dari `group_notebooks`; "Terakhir Diperbarui" = yang lebih baru dari `updated_at` dan kolom baru `study_groups.progress_updated_at` (ikut migrasi 9, belum di produksi), jadi hapus putaran pun memajukan tanggal. Pemeriksaan tetap `BELUM_DICEK` dan tidak ada yang Tuntas sampai 3C.
  - HTTP: `GET/POST /groups/{id}/submissions`, `PATCH/DELETE /submissions/{id}` (409 `NOT_LAST_ROUND`), `GET/PUT /groups/{id}/notebook`; kelompok bercatatan tidak bisa dihapus (409 `IN_USE`). Publik tidak memuat email/subjek/catatan (diuji).
  - **Uji mutan:** audit ditulis di luar transaksi dengan galat diabaikan → uji atomik gagal; lajur dari putaran terakhir saja → uji AC08 gagal.
  - OpenAPI 0.6.0: 29 jalur, skema `SubmissionRound(Write)`, `GroupNotebook`; semua `$ref` terselesaikan.
- **Task 3.8 selesai (15 Nov 2026).** Tes klien ditulis bersama kode, jadi **tidak ada RED ketat** yang diklaim. GREEN di `cbt-vps`: `npm ci`, **108 uji** Node (6 baru), lint 0 masalah, typecheck, build statis (rute baru `/admin/kelompok/detail/`).
  - Klien: `getGroup`, `listRounds`, `recordRound`, `updateRound`, `deleteRound`, `getNotebook`, `putNotebook`; validasi putaran sama dengan Go; konversi waktu WIB untuk `datetime-local` (`toWIBInput`/`fromWIBInput`, tanggal mustahil ditolak); `roundLabel` ("Pengumpulan awal", "Revisi n"); saran subjek juknis (`REVISI 1 - Proyek Informatika IX - 9A - Kelompok 1 - Struktur Data`); kode `NOT_LAST_ROUND` jadi pesan tetap.
  - UI `/admin/kelompok/detail/?id=` (tombol **Pengumpulan** di tiap baris kelompok): identitas kelompok, lajur Pengumpulan teks + ikon + warna, daftar putaran terbaru dulu (waktu WIB, PPT/PDF + nama berkas, pengirim, subjek, catatan guru), ubah di baris, hapus hanya putaran terakhir dengan konfirmasi; formulir **Catat Pengumpulan** dengan nomor berikutnya otomatis, waktu bawaan sekarang (WIB), pengirim bawaan email kelompok, tombol isi subjek juknis, centang PPT/PDF (nama berkas muncul hanya bila dicentang). Bagian **Buku tulis**: centang diterima + tanggal + catatan. Konflik, "bukan terakhir", dan "tidak ditemukan" memuat ulang data.
  - **E2E nyata di `cbt-vps`** (biner asli, database uji, klien asli): BELUM → PPT saja SEBAGIAN (AC02) → PDF LENGKAP (AC03) → revisi PPT saja tetap LENGKAP (AC08); klik ganda `conflict`; hapus putaran pertama `not_last`; hapus terakhir; ubah putaran 0 tambah PDF; ubah basi `conflict`; buku tulis diterima → publik `SUDAH_DITERIMA`; hapus kelompok bercatatan `in_use`; publik tanpa email/catatan/nama berkas/subjek; halaman detail 200; audit 6 baris berurutan. Database uji dibersihkan.
  - **Belum:** tampilan di browser nyata (360 px, keyboard, `datetime-local` di HP) — Task 3.9.
- **Perbaikan 3B (15 Nov 2026): hapus kelompok dengan buku tulis kosong.** Temuan cek browser 3.9: setelah buku tulis dicentang lalu dibatalkan, kelompok tidak bisa dihapus (baris `group_notebooks` kosong + `RESTRICT`). RED: uji baru gagal dengan `still has records`. GREEN: `DeleteGroup` dalam satu transaksi membuang dulu baris buku "belum diterima, tanpa catatan" (`DeleteEmptyNotebook`), lalu menghapus kelompok; buku diterima, buku bercatatan, dan putaran tetap `ErrInUse` tanpa menyentuh baris buku. Seluruh suite 2 kali, `-race`, vet, build, gofmt hijau; mutan "hapus semua baris buku" tertangkap.
  - Dideploy ke produksi (rilis `20261115T125117Z-a2d9fc52cbba`, cadangan 44271 B). Data uji `UJI` + Kelompok 1 dihapus lewat UI; audit tercatat; produksi kembali kosong.
- **Perbaikan 360 px halaman detail (18 Nov 2026).** Edge headless 360 px dengan API tiruan menemukan meluap 491 px (email dan nama berkas panjang). Perbaikan CSS `77788ad` dideploy (rilis `20261118T045543Z-cffeaffbd4f3`); produksi 360 px, 0 elemen meluap.
- **3B ditutup 18 Nov 2026** atas instruksi user ("Tutup Fase 3B dengan pengecualian"), dengan pengecualian di `docs/F3B_ACCEPTANCE.md`: keyboard halaman detail, `datetime-local` di HP, AC08 langsung di produksi, uji komponen UI, dan kunci `FOR UPDATE` trigger.
