# Fase 3 — Modul Progres Tugas Kelompok (Rencana)

> Status: **DISETUJUI 14 Nov 2026** (user mengikuti semua rekomendasi). 3A dimulai.
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
| 3.4 | UI admin kelas/kelompok; `/progres/` memuat tabel, filter, cari, keadaan kosong/galat | klien API diuji; lint/typecheck/build |
| 3.5 | Deploy + cek browser + `docs/F3A_ACCEPTANCE.md` (AC01, AC04 sebagian, AC18, AC20) | semua gerbang |

### 3B — Pengumpulan dan buku tulis

| Task | Isi | Uji |
|---|---|---|
| 3.6 | Migrasi `submission_rounds`, `group_notebooks` | uji skema: unik `(group, round)`, minimal satu berkas per putaran |
| 3.7 | Status pengumpulan terhitung; catat/ubah/hapus putaran; buku tulis; audit dalam transaksi | TDD: AC02, AC03, AC08; klik ganda 409 |
| 3.8 | UI admin "Catat Pengumpulan" + buku tulis di detail kelompok; publik menampilkan status | lint/typecheck/build + E2E |
| 3.9 | Deploy + cek browser + `docs/F3B_ACCEPTANCE.md` | semua gerbang |

### 3C — Pemeriksaan

| Task | Isi | Uji |
|---|---|---|
| 3.10 | Migrasi `group_reviews`, `evaluation_findings` | uji skema |
| 3.11 | Aturan status review + temuan draf/terbit/selesai/arsip + TUNTAS | TDD: AC05, AC09, AC10; transisi ilegal ditolak |
| 3.12 | UI admin evaluasi: tambah temuan, terbitkan dengan konfirmasi, ubah status | lint/typecheck/build + E2E |
| 3.13 | Deploy + cek browser + `docs/F3C_ACCEPTANCE.md` | semua gerbang |

### 3D — Portal kelompok

| Task | Isi | Uji |
|---|---|---|
| 3.14 | Migrasi `group_access_sessions`; kode akses HMAC; rotasi mencabut sesi | TDD: kode tidak tersimpan mentah; rotasi |
| 3.15 | `group/access`, `group/me`, `group/logout`; rate limit; identitas hanya dari sesi | TDD: AC06, AC07 (manipulasi ID/URL/cookie), AC13, AC24 |
| 3.16 | UI `/progres/masuk/` dan `/progres/kelompok/`; admin "Buat/ganti kode" dengan tampilan sekali | lint/typecheck/build + E2E |
| 3.17 | `GROUP_CODE_KEY` di `.env` + `deploy.sh` (izin), deploy, cek browser, `docs/F3D_ACCEPTANCE.md` | semua gerbang |

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
