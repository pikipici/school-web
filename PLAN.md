# PLAN.md — Website Monitoring dan Evaluasi Tugas Kelompok Informatika Kelas IX

**Status:** Spesifikasi pengembangan / developer handoff
**Versi:** 2.1 — 13 November 2026 (revisi arah: web pribadi guru dengan Pengumuman dan Materi)
**Tahun pelajaran awal:** 2026/2027
**Bahasa antarmuka:** Bahasa Indonesia
**Prinsip utama:** Pengumpulan tugas tetap melalui email guru. **Website TIDAK terintegrasi dengan Gmail; seluruh status, pemeriksaan, dan revisi dicatat MANUAL oleh guru.** Backend aplikasi wajib menggunakan **Go**, sedangkan Next.js hanya untuk frontend.

---

## 0. Revisi arah produk (13 November 2026)

Produk bukan lagi alat tunggal monitoring, melainkan **web pribadi guru** yang memuat konten sekolah. Monitoring tugas kelompok (bagian 1–12 di bawah) tetap berlaku utuh, tetapi menjadi **satu modul** bernama *Progres Tugas Kelompok*. Rincian keputusan: `.kiro/steering/REDESIGN_BACKLOG.md`; arah visual: `DESIGN.md`.

| Modul | Isi | Siapa yang menulis | Akses baca |
|---|---|---|---|
| **Pengumuman** | Judul, tanggal (WIB), isi singkat | Guru lewat dashboard admin | Publik |
| **Materi** | Judul, topik, deskripsi, lampiran | Guru lewat dashboard admin | Publik |
| **Progres Tugas Kelompok** | Seluruh isi PLAN ini | Guru lewat dashboard admin | Publik ringkas; detail via kode akses kelompok |

Keputusan terkunci:

- Halaman depan langsung menampilkan konten (Pengumuman terbaru, lalu daftar Materi), tanpa hero.
- Menu publik: `Pengumuman`, `Materi`, `Progres Tugas`.
- Header menampilkan **nama pribadi guru saja**, tanpa foto. Nama: "Fikri Rama Aulia" (`frontend/src/lib/site.ts`).
- Konten Pengumuman dan Materi ditulis lewat dashboard admin Go; tidak ada penulisan konten dari browser publik.
- **Lampiran Materi dari guru** diizinkan dengan tipe **PDF dan PPT/PPTX** saja. Ini pengecualian terbatas atas larangan upload di bawah: larangan tetap berlaku untuk **kelompok/siswa** dan untuk berkas tugas.
- Ketentuan lampiran yang harus diselesaikan sebelum implementasi (belum diputuskan): ukuran maksimal per berkas, lokasi penyimpanan (disk VPS di luar direktori publik atau layanan lain), validasi tipe berdasarkan isi bukan hanya ekstensi, nama berkas aman, dan jalur unduh lewat Go.
- Pengumuman dan Materi baru berisi data setelah login admin dan database tersedia (Fase 1 dan sesudahnya). Sampai itu, tampilan menunjukkan keadaan kosong yang jujur.

---

## 1. Ringkasan dan tujuan

Bangun website responsif untuk guru Informatika kelas IX yang menampilkan kelompok mana yang sudah/belum mengumpulkan tugas, status pemeriksaan, dan catatan kesalahan **spesifik per file serta letaknya**. Guru mengecek email dan file di luar website, lalu mengisi hasilnya lewat dashboard admin. Siswa membaca status serta catatan revisi melalui website tanpa mengharapkan balasan email guru.

Tujuan:

1. Memantau semua kelompok, termasuk yang belum mengumpulkan sama sekali.
2. Memisahkan status **pengumpulan**, **evaluasi**, dan **buku tulis**.
3. Guru dapat menulis catatan semisal `PPT, Slide 5: diagram kurang jelas; tambahkan keterangan setiap simpul` atau `PDF, Halaman 3: definisi Tree belum tepat; revisi definisi beserta contoh`.
4. Catatan dapat disimpan sebagai draf, lalu **dipublikasikan secara sengaja**.
5. Kelompok melihat evaluasinya sendiri, bukan detail kelompok lain.
6. Semua putaran revisi memiliki histori, tanggal, dan jejak tindakan guru.
7. Kewajiban pertanyaan saat kelompok lain presentasi dapat dicatat.

### Non-negotiable

- **Tidak ada** Gmail API, IMAP, OAuth akses Gmail, sinkronisasi inbox, atau balas email otomatis.
- **Tidak ada** unggah/penyimpanan PPT atau PDF **tugas kelompok** di website pada MVP. Pengecualian: lampiran Materi dari guru (lihat bagian 0).
- Website **tidak memverifikasi sendiri** email sudah masuk; sumber status adalah input guru.
- Siswa **tidak dapat** mengubah status atau mengesahkan revisinya sendiri.
- **Semua logika backend utama dan API bisnis wajib di Go**; jangan mengganti backend dengan Next.js Route Handlers/Server Actions, Supabase Edge Functions, atau implementasi CRUD langsung dari browser.

## 2. Aturan proyek pembelajaran yang dikunci

### 2.1 Tujuh materi

| Kode | Materi |
|---|---|
| K1 | Struktur Data (Graf dan Tree) |
| K2 | Ekspresi dan Operasi Logika, Algoritma Perulangan |
| K3 | Blogspot |
| K4 | Isu Keamanan Internet (Privasi, Cyberbullying, Hoaks) |
| K5 | Visualisasi Data |
| K6 | Keamanan Data dan Ancaman Digital (Phishing, Malware) |
| K7 | Pseudocode dan Flowchart |

Ketujuh topik menjadi *seed* awal. Website mendukung **beberapa kelas IX**, dengan kelompok per kelas, bukan terbatas tujuh kelompok total.

### 2.2 Tiga bentuk tugas wajib

1. **PPT** — satu file presentasi per kelompok, dikirim melalui email.
2. **PDF** — satu file modul materi per kelompok, dikirim melalui email.
3. **Buku tulis** — **cukup satu buku catatan per kelompok**, dikumpulkan **fisik** kepada guru; anggota kelompok tidak wajib memiliki buku catatan masing-masing.

### 2.3 Email resmi pengumpulan

- Guru: **fikriramaaulia@gmail.com**.
- Alamat yang harus dibuat kelompok: `[namakelompok].[kelas].2627.Informatika@gmail.com`.
- Contoh: `kelompok1.9a.2627.Informatika@gmail.com`.
- Contoh subjek awal: `Proyek Informatika IX - 9A - Kelompok 1 - Struktur Data`.
- Contoh subjek revisi: `REVISI 1 - Proyek Informatika IX - 9A - Kelompok 1 - Struktur Data`.
- **Admin memasukkan tanggal penerimaan, alamat pengirim, jenis file, dan nama file secara manual setelah membuka Gmail sendiri.**
- Jangan meminta, mengunggah, atau menyimpan kata sandi Gmail kelompok pada website.

Catatan pelaksanaan: pembuatan akun oleh siswa harus mengikuti kebijakan sekolah, persetujuan yang diperlukan, dan persyaratan usia penyedia layanan. Jika perlu akun sekolah terkelola, pengecualiannya harus ditetapkan guru/sekolah, bukan oleh aplikasi secara otomatis.

### 2.4 Pertanyaan antarkelompok

- Setiap kelompok wajib menyiapkan **minimal satu pertanyaan yang relevan** untuk setiap presentasi kelompok lain.
- Pertanyaan dicatat di satu buku kelompok dan diajukan saat sesi tanya jawab sesuai arahan guru.
- Guru dapat menandai kesiapan dan penyampaian pertanyaan secara manual.

### 2.5 Pembagian tugas anggota

Peran yang dapat dicatat secara opsional: (1) mencari/menyusun materi, (2) menyusun PPT, (3) menyusun PDF, (4) menulis buku catatan kelompok. Semua anggota tetap bertanggung jawab memahami materi.

---

## 3. Jenis pengguna dan izin

| Jenis | Bisa melakukan | Tidak boleh melakukan |
|---|---|---|
| **Publik/siswa tanpa login** | Melihat tabel seluruh kelompok: kelas, topik, status pengumpulan, status evaluasi ringkas, waktu pembaruan | Membaca catatan kesalahan detail, email pengirim, audit log, atau data privat |
| **Kelompok dengan kode akses** | Membaca detail status dan **catatan yang sudah dipublikasikan** milik kelompoknya sendiri, riwayat revisinya | Mengubah status/evaluasi, melihat detail kelompok lain, melihat draf guru |
| **Guru/admin** | Mengelola kelas/kelompok, mencatat kiriman, buku, evaluasi, revisi, pertanyaan, ekspor rekap | Menggunakan website untuk membaca email otomatis (fitur tidak ada) |

**Default akses:** monitoring publik tanpa akun; detail revisi dengan **kode akses kelompok yang kuat**. Guru memberikan kode privat kepada tiap kelompok. Jika suatu saat sekolah menghendaki detail juga publik, itu merupakan **perubahan kebijakan privasi** dan tidak termasuk MVP.

## 4. MVP: fitur wajib dan fitur yang sengaja dikecualikan

### Fitur wajib

- [ ] Pengumuman: guru membuat, mengubah, dan menghapus; publik membaca (lihat bagian 0).
- [ ] Materi: guru mengelola entri dan lampiran PDF/PPT; publik membaca dan mengunduh (lihat bagian 0).
- [ ] Login aman untuk guru/admin; tidak ada pendaftaran admin publik.
- [ ] CRUD kelas, kelompok, topik terpilih, email resmi kelompok, dan tahun pelajaran.
- [ ] Data awal tujuh topik dan tahun ajaran 2026/2027.
- [ ] Dashboard monitoring publik, ringkasan angka, filter kelas/status, pencarian.
- [ ] Pencatatan penerimaan PPT dan PDF secara **manual**, termasuk kiriman sebagian dan revisi.
- [ ] Pencatatan serah terima satu buku tulis kelompok secara **manual**.
- [ ] Form catatan perbaikan: file, lokasi (slide/halaman/bagian), masalah, perbaikan yang diharapkan, prioritas.
- [ ] Draf evaluasi, publikasi, dan status temuan terbuka/selesai.
- [ ] Portal kelompok privat dengan kode akses dan riwayat revisi.
- [ ] Pencatatan partisipasi pertanyaan antarkelompok.
- [ ] Rekap/ekspor CSV hanya untuk guru.
- [ ] Audit log perubahan penting.
- [ ] Desain responsif HP/laptop dan label berbahasa Indonesia.

### Bukan cakupan MVP

- Integrasi Gmail atau notifikasi email otomatis.
- Upload, preview, dan penyimpanan file PPT/PDF **tugas kelompok** dalam website (lampiran Materi dari guru dikecualikan).
- AI grading, OCR, atau pemeriksaan berkas otomatis.
- Forum/chat, akun tiap siswa, ranking publik, aplikasi Android, atau WhatsApp bot.
- Penghitungan nilai otomatis sebelum bobot penilaian dikonfirmasi guru.

---

## 5. Model status — sangat penting

**Jangan membuat satu status yang mencampur “terkumpul”, “diperiksa”, dan “selesai”.** Gunakan tiga indikator utama, dan satu status total yang diturunkan dari ketiganya.

### 5.1 Status pengumpulan PPT/PDF (dihitung sistem dari catatan guru)

| Kode | Label | Kondisi |
|---|---|---|
| `BELUM` | Belum Mengumpulkan | Tidak ada PPT ataupun PDF yang ditandai diterima |
| `SEBAGIAN` | Terkumpul Sebagian | Baru PPT **atau** PDF yang diterima |
| `LENGKAP` | Berkas Digital Lengkap | PPT **dan** PDF sudah diterima |

Revisi satu jenis file **tidak menghapus** jenis file sebelumnya yang masih berlaku. Setiap versi file tetap tercatat.

### 5.2 Status evaluasi (ditentukan guru)

| Kode | Label |
|---|---|
| `BELUM_DICEK` | Belum Diperiksa |
| `DIPERIKSA` | Sedang Diperiksa |
| `PERLU_REVISI` | Perlu Revisi |
| `REVISI_DITERIMA` | Revisi Diterima, Menunggu Pemeriksaan |
| `DISETUJUI` | Materi Disetujui |

Aturan validasi:

- `PERLU_REVISI` baru boleh dipublikasikan jika ada **minimal satu catatan wajib perbaikan** yang dipublikasikan.
- `DISETUJUI` hanya bila PPT dan PDF sudah lengkap serta **tidak ada catatan wajib yang masih terbuka**.
- Revisi yang masuk menambah **putaran pengumpulan baru**, bukan menimpa bukti lama.
- Hanya admin yang bisa mengubah status.

### 5.3 Buku tulis

`BELUM_DITERIMA` atau `SUDAH_DITERIMA`, dengan tanggal diterima dan catatan opsional. Status keseluruhan **`TUNTAS`** hanya jika berkas digital lengkap, evaluasi disetujui, **dan** buku tulis diterima. Kewajiban bertanya dilacak terpisah agar tidak membingungkan dengan penerimaan tugas.

### 5.4 Tampilan publik

Gunakan kolom terpisah: **Kelompok | Kelas | Topik | Pengumpulan | Pemeriksaan | Buku Tulis | Terakhir Diperbarui**. Jangan tampilkan informasi internal, alamat email kelompok, atau detail kesalahan ke pengunjung publik.

---

## 6. Alur operasional nyata

### A. Guru menyiapkan proyek

1. Login ke dashboard admin.
2. Buat kelas (`9A`, `9B`, dst.) dan kelompoknya.
3. Pilih satu dari tujuh topik untuk masing-masing kelompok.
4. Catat akun email kelompok sesuai format juknis.
5. Hasilkan kode akses kuat per kelompok dan distribusikan **secara privat**.
6. Semua kelompok langsung muncul di monitoring dengan status **Belum Mengumpulkan**.

### B. Kelompok mengirim tugas

1. Kelompok mengirim PPT dan PDF ke email guru dari akun kelompok.
2. Guru mengecek kotak masuk Gmail **secara manual di luar website**.
3. Admin memilih kelompok lalu klik **Catat Pengumpulan**.
4. Isi tanggal/waktu email diterima, alamat pengirim, subjek (opsional), centang `PPT` dan/atau `PDF`, nama file opsional, serta catatan guru.
5. Website memperbarui status pengumpulan. Guru mencatat buku tulis jika sudah diserahkan fisik.

### C. Guru memeriksa dan memberi revisi

1. Guru memeriksa file dari Gmail atau salinan lokal.
2. Di website, buka **Evaluasi Kelompok → Tambah Temuan**.
3. Pilih `PPT`/`PDF`/`Buku`/`Presentasi`, tulis **lokasi spesifik**, deskripsi kesalahan, dan instruksi perbaikan.
4. Simpan sebagai **Draf** (belum terlihat siswa).
5. Klik **Publikasikan Evaluasi** jika sudah siap.
6. Tetapkan `Perlu Revisi` bila ada temuan wajib.
7. Kelompok membaca catatan yang dipublikasikan melalui portalnya.

### D. Perbaikan dan pengumpulan ulang

1. Kelompok memperbaiki file dan mengirim email baru ber-subjek `REVISI 1 - ...`, `REVISI 2 - ...`, dst.
2. Guru memeriksa Gmail di luar website.
3. Guru menambahkan putaran revisi baru secara manual; jika hanya PPT direvisi, PDF sebelumnya tetap tercatat.
4. Guru mengubah status menjadi `Revisi Diterima`, kemudian `Sedang Diperiksa`.
5. Guru menandai temuan yang sudah selesai, menambah temuan jika perlu, lalu memublikasikan hasil berikutnya.
6. Jika tidak ada masalah wajib, guru menandai `Materi Disetujui`.
7. Jika buku tulis juga diterima, status keseluruhan menjadi `Tuntas`.

### E. Presentasi kelompok lain

1. Guru mencatat sesi presentasi kelompok penyaji.
2. Untuk tiap kelompok lainnya pada kelas yang sama, guru bisa mencatat apakah telah menyiapkan dan/atau mengajukan pertanyaan relevan.
3. Rekap per sesi menampilkan kelompok yang sudah dan belum memenuhi kewajiban.

---

## 7. Halaman website dan kebutuhan UI

| Route | Pengguna | Isi utama |
|---|---|---|
| `/` | Publik | Ringkasan proyek dan tombol monitoring / akses kelompok |
| `/monitoring` | Publik | Statistik, tabel semua kelompok, filter dan pencarian |
| `/juknis` | Publik | Ringkasan ketentuan dan tautan juknis resmi jika ada |
| `/akses-kelompok` | Kelompok | Form memasukkan kode akses |
| `/kelompok/saya` | Kelompok tervalidasi | Status, buku tulis, catatan evaluasi terpublikasi, riwayat revisi |
| `/admin/login` | Guru | Login |
| `/admin` | Guru | KPI dan daftar tindakan yang harus ditangani |
| `/admin/kelas` | Guru | CRUD kelas |
| `/admin/kelompok` | Guru | Daftar dan CRUD kelompok, pencarian/filter |
| `/admin/kelompok/[id]` | Guru | Detail dan ringkasan kelompok |
| `/admin/kelompok/[id]/pengumpulan` | Guru | Form kiriman awal/revisi & buku tulis |
| `/admin/kelompok/[id]/evaluasi` | Guru | Editor temuan, draf, publikasi, status |
| `/admin/partisipasi` | Guru | Sesi presentasi dan pertanyaan antarkelompok |
| `/admin/rekap` | Guru | Rekap, filter, unduh CSV |
| `/admin/audit` | Guru | Log perubahan |

**UX wajib:** mobile-first; label eksplisit; status punya teks dan ikon (bukan warna saja); form jelas; tombol cepat; konfirmasi sebelum publikasi/rotasi kode; keadaan loading, kosong, dan error; timestamp tampil **WIB (Asia/Jakarta)** dan disimpan sebagai UTC.

---

## 8. Arsitektur dan pilihan teknologi (dikunci: backend Go)

### 8.1 Pemisahan tanggung jawab

| Lapisan | Teknologi | Tanggung jawab |
|---|---|---|
| **Frontend** | **Next.js App Router + TypeScript + Tailwind CSS** | Halaman publik, dashboard guru, portal kelompok, state UI, validasi untuk pengalaman pengguna, memanggil API Go |
| **Backend utama** | **Go (`net/http` + `go-chi/chi/v5`)** | REST API, aturan bisnis, seluruh validasi tepercaya, login/sesi, otorisasi, CRUD, status, evaluasi, audit, ekspor CSV |
| **Database** | **PostgreSQL** (dapat memakai Supabase **sebagai hosting database saja**) | Menyimpan kelas, kelompok, metadata pengumpulan, catatan evaluasi, akun/sesi, dan riwayat |
| **Akses DB Go** | **pgx/v5 + pgxpool**, **sqlc** | Koneksi pool PostgreSQL, query SQL bertipe, transaksi |
| **Migrasi database** | **goose** (atau tool migrasi Go sekelas yang dipilih saat setup) | Skema database versioned dan reproducible |
| **Validasi input** | **Go structs + validasi di service/handler**; Zod opsional di frontend | Sumber kebenaran tetap validasi backend, bukan validasi browser |
| **Otentikasi guru** | **Go-managed admin session**, kata sandi hash **Argon2id** | Login admin, logout, otorisasi server-side; tidak ada daftar akun admin publik |
| **Otentikasi kelompok** | Kode akses acak kuat + **sesi Go HttpOnly** | Akses detail kelompok sendiri saja, tidak dapat mengubah data |
| **Testing** | `go test`, `httptest`, integrasi PostgreSQL; Vitest opsional; Playwright E2E | Pengujian backend, akses ilegal, perilaku UI, end-to-end |
| **Deployment** | **Single VPS `cbt-vps` + systemd**; Next.js static export dilayani binary Go; PostgreSQL di VPS | Satu origin publik, satu runtime Go, build/test/deploy di VPS mengikuti pola LMS |

**Keputusan eksplisit:** Next.js tidak boleh langsung mengakses tabel database, memakai Supabase SDK untuk query bisnis dari browser, atau menjalankan logika bisnis pada Server Actions. Semua permintaan data berjalan melalui API Go.

### 8.2 Target deployment yang dikunci

- SSH alias: `cbt-vps` (`ubuntu@43.134.111.212`).
- Checkout: `/home/ubuntu/informatika-monitoring`.
- Bare Git repo: `/home/ubuntu/git-repos/informatika-monitoring.git`.
- Service: `informatika-monitoring-api.service`.
- Port internal: `8300`.
- Database awal: `informatika_monitoring`, memakai role aplikasi khusus dan bukan superuser.
- Local Windows digunakan untuk coding + Git; install/build/test/migrate/runtime dilakukan di `cbt-vps`.
- Delivery mengikuti dual push GitHub `origin` + bare remote `server`, lalu checkout VPS di-fast-forward ke commit terverifikasi.
- Deploy script wajib melakukan pre-flight, test/build, migration Goose, backup binary, atomic swap, restart systemd, retry `/readyz`, dan rollback saat gagal.
- Source sync berbeda dari runtime activation: restart/deploy hanya dilakukan setelah checkpoint fase disetujui.

### 8.3 Diagram arsitektur

```text
                  +---------------------------+
                  |  Browser Guru / Siswa     |
                  +-------------+-------------+
                                |
                         HTTPS / Website
                                |
                  +-------------v-------------+
                  | Go binary + systemd        |
                  | - static Next.js export    |
                  | - /api/v1/* (chi + REST)   |
                  | - auth, izin, validasi,    |
                  |   status, evaluasi, audit  |
                  +-------------+-------------+
                                |
                         localhost / pgxpool
                                |
                  +-------------v-------------+
                  | PostgreSQL di cbt-vps      |
                  +---------------------------+

  Gmail Guru <--- siswa mengirim PPT/PDF melalui email
       |
       +--- Guru mengecek sendiri, lalu memasukkan status di UI

  TIDAK ADA jalur sinkronisasi dari Gmail ke website/Go.
```

### 8.4 Jaringan dan cookie

- Gunakan **satu origin publik**. Binary Go pada `cbt-vps` melayani static export Next.js dan `/api/v1/*` pada port internal `8300`; HTTPS/tunnel publik dapat ditambahkan di depan service tanpa memindahkan logika bisnis dari Go.
- Pada development, Next.js boleh memakai rewrite transparan `/api/*` ke Go. Pada production, browser berbicara ke origin yang sama dan static frontend tidak memiliki runtime server Next.js.
- Semua lalu lintas browser produksi memakai HTTPS. Cookie sesi `HttpOnly`, `Secure`, `SameSite=Lax/Strict` sesuai alur; validasi CSRF untuk request mutasi berbasis cookie dan periksa `Origin`.
- Layanan Go mengakses PostgreSQL lokal melalui credential khusus aplikasi dengan privilege minimum. Koneksi **tidak** berasal dari browser. Jika database dipindah ke managed service, koneksi wajib TLS.
- Data publik boleh di-cache singkat dan diinvalidasi saat berubah; **data admin/kelompok tidak boleh di-cache secara publik**.
- Sediakan `/healthz` untuk liveness dan `/readyz` untuk kesiapan database; jangan membeberkan secret dalam response/log.

### 8.5 Kebijakan pemilihan pustaka

- Gunakan rilis stabil Go dan dependency yang kompatibel saat implementasi, hindari versi hardcode tanpa alasan.
- `chi` dipilih karena ringan dan mengikuti `net/http`; `pgx` cocok untuk PostgreSQL; `sqlc` menghasilkan kode Go bertipe dari query SQL.
- Backend harus bisa berjalan dengan `go run ./cmd/api` saat development dan binary/Docker saat produksi.
- **Tidak perlu** Supabase Auth, Supabase Edge Functions, Firebase, Prisma, atau Next.js API untuk kebutuhan inti MVP.

Dokumentasi rujukan:
- https://go.dev/doc/
- https://pkg.go.dev/net/http
- https://github.com/go-chi/chi
- https://github.com/jackc/pgx
- https://docs.sqlc.dev/
- https://pressly.github.io/goose/
- https://nextjs.org/docs/app
- https://supabase.com/docs/guides/database/connecting-to-postgres

---

## 9. Desain database

Semua entitas utama memakai UUID, `created_at` dan `updated_at` bila sesuai. Simpan skema sebagai migrasi SQL. Nama enum/status harus konsisten dengan Bagian 5.

| Tabel | Kolom utama | Aturan |
|---|---|---|
| `academic_years` | `id`, `label`, `is_active` | Awal: `2026/2027`; dapat ditambah tahun lain |
| `classes` | `id`, `academic_year_id`, `code`, `name` | Unik `(academic_year_id, code)` |
| `topics` | `id`, `code`, `title`, `description`, `sort_order` | Seed tujuh topik K1–K7 |
| `groups` | `id`, `class_id`, `topic_id`, `name`, `slug`, `expected_email`, `access_code_digest`, `is_active` | Unik `(class_id, slug)`; akses hanya dengan kode acak kuat |
| `submission_rounds` | `id`, `group_id`, `round_number`, `received_at`, `sender_email`, `email_subject`, `ppt_received`, `pdf_received`, `ppt_filename`, `pdf_filename`, `internal_note`, `recorded_by` | Putaran 0 = awal; 1+ = revisi; unik `(group_id, round_number)` |
| `group_notebooks` | `id`, `group_id`, `is_received`, `received_at`, `note`, `updated_by` | Satu record buku fisik per kelompok |
| `group_reviews` | `id`, `group_id`, `review_status`, `summary`, `revision_due_at`, `last_published_at`, `updated_by` | Status pemeriksaan dipilih admin |
| `evaluation_items` | `id`, `group_id`, `submission_round_id`, `file_type`, `location_label`, `issue_description`, `required_action`, `priority`, `resolution_status`, `is_published`, `published_at`, `resolved_at`, `created_by` | Draf tidak terlihat kelompok; lokasi dan tindakan wajib untuk publikasi |
| `presentation_sessions` | `id`, `class_id`, `presenting_group_id`, `scheduled_at`, `actual_at` | Sesi per kelas |
| `question_participation` | `id`, `session_id`, `asking_group_id`, `question_text`, `is_prepared`, `was_asked`, `teacher_note` | Kelompok bertanya tidak boleh sama dengan penyaji; harus sekelas |
| `admin_users` | `id`, `email`, `password_hash`, `display_name`, `role`, `is_active` | Akun dibuat/direset melalui perintah CLI terproteksi; `password_hash` Argon2id; tanpa pendaftaran publik |
| `admin_sessions` | `id`, `admin_user_id`, `token_digest`, `expires_at`, `revoked_at`, `created_at` | Sesi guru server-side di Go; simpan digest token bukan cookie mentah |
| `audit_logs` | `id`, `actor_admin_id`, `entity_type`, `entity_id`, `action`, `before_json`, `after_json`, `created_at` | Tidak bisa diubah lewat UI biasa |
| `group_access_sessions` | `id`, `group_id`, `token_digest`, `expires_at`, `revoked_at` | Hanya jika memakai sesi kelompok server-side |

### Aturan data yang sering terlewat

1. Saat kiriman revisi **hanya berisi PPT**, status penerimaan PDF sebelumnya **tetap ada**. Hitung ketersediaan file dari seluruh kiriman valid, dan tunjukkan versi terbaru tiap jenis.
2. Jangan menyimpan file asli atau konten Gmail di database MVP; simpan **metadata yang diinput guru** saja.
3. Perubahan status, penerimaan kiriman, penerbitan evaluasi, dan reset kode harus meninggalkan audit log.
4. Evaluasi terpublikasi tidak dihapus permanen dengan mudah; gunakan arsip/soft delete jika koreksi diperlukan.
5. Pastikan `submission_rounds` mendukung kiriman hanya PPT, hanya PDF, atau keduanya.
6. Pada pengiriman pertama, default status review `BELUM_DICEK`; jangan otomatis `DISETUJUI`.
7. Gunakan indeks pada `groups(class_id)`, `submission_rounds(group_id, round_number)`, `evaluation_items(group_id, is_published)`, sesi aktif, dan log timestamp.
8. Semua operasi yang harus konsisten (mis. penambahan putaran revisi + audit log, publikasi evaluasi + perubahan status) menggunakan **transaksi PostgreSQL** di Go.
9. `admin_users.id` direferensikan oleh `recorded_by`, `updated_by`, dan `actor_admin_id` sesuai aturan foreign key. Database bukan tempat mengidentifikasi akun Gmail; itu tetap hanya metadata pengumpulan manual.

---

## 10. Kontrak API REST — seluruh endpoint ditangani Go

API berbasis JSON pada `/api/v1/` kecuali endpoint unduh CSV. **Tidak ada** Server Actions/Route Handlers Next.js untuk logika bisnis. Next.js hanya menjadi pemanggil API atau proxy HTTP transparan.

### 10.1 Autentikasi Go

| Method | Endpoint | Akses | Perilaku |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | Publik | Verifikasi email/password admin dengan hash Argon2id; rate limit; buat sesi HttpOnly jika benar |
| `POST` | `/api/v1/auth/logout` | Admin | Cabut sesi guru di database dan hapus cookie |
| `GET` | `/api/v1/auth/me` | Admin | Tampilkan identitas admin aktif tanpa password/hash |
| `POST` | `/api/v1/group/access` | Publik | Validasi kode akses, rate limit, terbitkan sesi privat kelompok |
| `POST` | `/api/v1/group/logout` | Kelompok | Cabut sesi kelompok |
| `GET` | `/api/v1/group/me` | Kelompok | Status dan evaluasi **hanya untuk kelompok sesi aktif**, hanya yang dipublikasikan |

**Provisioning admin:** gunakan perintah CLI Go (`go run ./cmd/admin create`) atau seed terproteksi pada deploy pertama; kata sandi disuplai dari prompt/env rahasia sekali pakai, bukan dikomit di Git. Implementasikan rotasi/reset aman untuk guru.

### 10.2 Publik — hanya data aman

- `GET /api/v1/public/summary?classId=...`
- `GET /api/v1/public/groups?classId=...&status=...&q=...`
- `GET /api/v1/public/topics`

### 10.3 Admin — seluruhnya membutuhkan sesi/role guru

- `GET/POST/PATCH/DELETE /api/v1/admin/classes` dan rute detail.
- `GET/POST/PATCH/DELETE /api/v1/admin/groups` dan rute detail.
- `POST /api/v1/admin/groups/{id}/rotate-access`.
- `POST /api/v1/admin/groups/{id}/submissions` (catatan email yang sudah diperiksa **manual**).
- `PATCH /api/v1/admin/groups/{id}/notebook`.
- `POST /api/v1/admin/groups/{id}/evaluations`.
- `PATCH /api/v1/admin/evaluations/{id}`.
- `POST /api/v1/admin/groups/{id}/publish-review`.
- `PATCH /api/v1/admin/groups/{id}/review-status`.
- CRUD `/api/v1/admin/presentations` dan `/api/v1/admin/question-participation`.
- `GET /api/v1/admin/reports/export.csv`.
- `GET /api/v1/admin/audit`.

### 10.4 Standar request/response

- Input/output JSON memakai penamaan konsisten (`camelCase` pada JSON). Field wajib divalidasi di Go, termasuk ukuran payload, UUID, tanggal, panjang string, status, perizinan, dan transisi status.
- Gunakan status HTTP standar (`200`, `201`, `204`, `400`, `401`, `403`, `404`, `409`, `422`, `429`, `500`).
- Bentuk error: `{"error":{"code":"VALIDATION_ERROR","message":"Lokasi kesalahan wajib diisi."}}`; jangan mengirim stack trace atau kredensial.
- List yang berpotensi besar menerima `page`/`limit`, batas maksimum; query DB terparameterisasi melalui `sqlc`/`pgx`.
- Berikan **idempotency key** atau perlindungan duplikasi pada input penerimaan/publikasi yang dapat terkirim dua kali (mis. klik ganda / retry jaringan).
- Semua aksi mutasi dilindungi middleware `requireAdmin`, CSRF check, validasi resource, dan audit; **semua keputusan izin tetap di Go**.
- Siapkan dokumentasi OpenAPI (`docs/openapi.yaml`) agar kontrak frontend–backend jelas dan bisa dites.

### 10.5 Contoh response aman untuk portal kelompok

```json
{
  "group": {"name": "Kelompok 1", "classCode": "9A", "topic": "Struktur Data"},
  "digitalSubmissionStatus": "LENGKAP",
  "reviewStatus": "PERLU_REVISI",
  "notebookReceived": false,
  "publishedFindings": [
    {
      "fileType": "PPT",
      "location": "Slide 5",
      "problem": "Diagram belum diberi label",
      "requiredAction": "Tambahkan nama simpul dan keterangan hubungan",
      "status": "TERBUKA"
    }
  ],
  "lastUpdatedAt": "2026-11-12T07:00:00Z"
}
```

Objek contoh **dummy** untuk pengujian, bukan catatan kelompok sungguhan. Jangan sertakan email lengkap, catatan internal, draft, token sesi, atau hash akses pada response.

---

## 11. Keamanan dan privasi (wajib untuk backend Go)

- **Jangan membuat koneksi Gmail apa pun.** Tidak boleh ada token/izin Gmail, OAuth inbox, IMAP, pengiriman email otomatis, atau penyimpanan password Gmail.
- **Wajib otorisasi Go pada setiap request:** handler publik hanya membaca data yang aman; endpoint admin memeriksa sesi dan role; endpoint kelompok mengambil `group_id` dari sesi, bukan dari ID bebas yang dikirim browser.
- **Login guru:** hash password dengan Argon2id dan salt unik; bandingkan secara aman; gunakan sesi opaque acak (bukan JWT statis yang sulit dicabut), simpan hanya digest token sesi; rate limit login; logout serta revocation.
- **Kode kelompok:** generator kriptografis (>=128-bit entropi), hash/HMAC aman di database, rotasi dan revocation sesi; rate limit percobaan serta pesan kesalahan generik untuk mencegah enumerasi kelompok.
- **Cookie:** `Secure`, `HttpOnly`, `SameSite=Lax/Strict`, path yang tepat, expiry; request mutasi berbasis cookie memerlukan perlindungan CSRF dan pemeriksaan Origin; batasi percobaan berulang.
- **Database:** Go memegang satu-satunya credential koneksi DB pada server; pakai role DB privilege minimum, TLS, pool connection terbatas, SQL terparameterisasi, migrasi terkontrol. Jika memakai Supabase Postgres, **jangan** membiarkan client/browser mengakses tabel privat via Data API/anon key; disable/expose secara aman, dan jangan menganggap RLS otomatis berlaku untuk koneksi privileged Go.
- **Validasi:** batas ukuran body, timeout HTTP, validasi struktur dan panjang field, normalisasi data, sanitasi output yang diperlukan; hindari raw HTML yang merender konten guru/siswa.
- **Privasi:** monitoring publik tidak boleh mengirim email kelompok, nama anggota, catatan detail, nama file, draft guru, audit, atau metadata sesi. Draf dan komentar internal hanya bisa dilihat guru.
- **Audit:** seluruh aksi perubahan status, penerimaan, publikasi, reset kode, dan perubahan akun tercatat; tidak memasukkan token, kode akses, password, atau isi cookie dalam log.
- **Jaringan:** HTTPS pada frontend/API, same-origin proxy dianjurkan, CORS dibatasi jika diperlukan; jangan menganggap CORS mekanisme autentikasi.
- **Operasional:** backup dan restore diuji; pertimbangkan backup otomatis penyedia database; log terstruktur tanpa data sensitif; rate limiting dan batas koneksi; rahasia lewat environment/secret store.
- **Cache:** detail admin/kelompok `Cache-Control: no-store`, noindex untuk halaman privat. Public status boleh memakai cache terkontrol dengan invalidasi.
- **Akses admin awal:** tidak ada registrasi publik; buat akun melalui CLI terproteksi, gunakan password kuat dan prosedur reset aman; MFA adalah peningkatan lanjutan.

**Kasus uji wajib:** kelompok A mengubah URL/query menjadi kelompok B; mencoba kode akses berkali-kali; mendapatkan draf melalui request langsung; POST status tanpa cookie admin; cookie yang dicabut; CSRF dari origin asing; CSV tanpa admin; password/secret bocor melalui response/log.

---

## 12. Struktur proyek monorepo (frontend Next.js + backend Go)

```text
informatika-monitoring/
├── PLAN.md
├── README.md
├── .gitignore
├── docker-compose.yml                  # PostgreSQL development + layanan jika dibutuhkan
├── Makefile                            # dev, test, migrate, build
├── frontend/                           # HANYA UI Next.js
│   ├── package.json
│   ├── next.config.ts                   # rewrite /api/* ke Go bila satu origin
│   ├── .env.example                     # hanya nilai frontend non-rahasia
│   ├── src/
│   │   ├── app/
│   │   │   ├── (public)/
│   │   │   ├── monitoring/page.tsx
│   │   │   ├── akses-kelompok/page.tsx
│   │   │   ├── kelompok/saya/page.tsx
│   │   │   └── admin/
│   │   ├── components/{ui,admin,public,groups}/
│   │   ├── lib/api-client/             # pemanggil endpoint Go
│   │   └── types/
│   └── tests/
├── backend/                            # SELURUH API & LOGIKA BISNIS DALAM GO
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   ├── .env.example                     # DATABASE_URL dan sesi rahasia (nilai kosong)
│   ├── cmd/
│   │   ├── api/main.go                 # start HTTP server
│   │   └── admin/main.go               # create/reset akun guru
│   ├── internal/
│   │   ├── config/
│   │   ├── http/
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── router.go
│   │   ├── auth/                       # password, sessions, group tokens
│   │   ├── service/                    # business rules, transaction boundary
│   │   ├── repository/                 # DB queries via sqlc
│   │   ├── db/                         # pool & generated sqlc code
│   │   └── domain/                     # entity & status
│   ├── sql/
│   │   ├── migrations/                # goose SQL migrations
│   │   ├── queries/                   # sqlc query source
│   │   └── seed.sql                   # topik K1–K7 (tanpa siswa nyata)
│   ├── sqlc.yaml
│   └── tests/
│       ├── integration/
│       └── e2e/
└── docs/
    ├── openapi.yaml
    ├── DEPLOYMENT.md
    ├── ADMIN_GUIDE.md
    └── SECURITY.md
```

**Aturan struktur:** frontend tidak boleh memiliki koneksi SQL, secret DB, penulisan status via Server Actions, maupun handler API bisnis; semua berada di `backend/` Go. Gunakan satu repository supaya kontrak endpoint, dokumentasi, dan pengujian mudah dijaga.

---

## 13. Roadmap eksekusi berfase (Go-first)

### Fase 0 — Inisialisasi monorepo

**Tugas:** buat `frontend/` Next.js + TypeScript + Tailwind dan `backend/` Go; router `chi`, konfigurasi environment, `.gitignore`, Makefile, Docker/PostgreSQL lokal, `/healthz`; tulis kontrak awal OpenAPI.
**Done jika:** `go run ./cmd/api` dan `npm run dev` berjalan; frontend dapat memanggil `GET /api/v1/public/summary` melalui proxy; semua rahasia tetap di server.

### Fase 1 — Database, migrasi, dan login guru di Go

**Tugas:** goose migrations, sqlc + pgxpool, seed tujuh topik, tahun ajaran; akun guru via CLI; hash Argon2id, sesi Go, autentikasi, rate limit, middleware admin, audit dasar; uji akses DB privat.
**Done jika:** endpoint login/logout admin berfungsi, sesi dicabut setelah logout, pengguna tak berwenang tidak bisa membaca data privat/mengubah status, dan **Next.js tidak mengakses database langsung**.

### Fase 2 — CRUD kelas/kelompok + monitoring publik

**Tugas:** Go API CRUD dan endpoint publik; frontend admin kelas/kelompok, status awal, tabel/filter/pencarian dan statistik; hitung status server-side dari catatan.
**Done jika:** kelompok baru tampil sebagai `Belum Mengumpulkan` walau tidak ada email masuk; publik tidak melihat data privat.

### Fase 3 — Input pengumpulan manual

**Tugas:** Go endpoint catat PPT/PDF per putaran, metadata email dan waktu, pencatatan buku tulis, transaksi dan audit; frontend form input yang jelas.
**Done jika:** PPT saja menghasilkan `Sebagian`, disusul PDF menjadi `Lengkap`; buku tulis tetap indikator terpisah; semua input dilakukan guru.

### Fase 4 — Evaluasi dan publikasi

**Tugas:** service Go menangani draf/terbit, temuan per jenis file/slide/halaman, instruksi, validasi transisi; UI admin untuk catatan; transaksi dan audit.
**Done jika:** draf tidak tampil ke kelompok; catatan detail hanya muncul setelah dipublikasikan oleh guru.

### Fase 5 — Akses privat kelompok + riwayat revisi

**Tugas:** kode akses aman dan sesi server-side Go, rotasi dan rate limit; endpoint hanya mengambil identitas kelompok dari sesi; histori revisi per file; UI portal siswa.
**Done jika:** kelompok A tidak dapat mengakses temuan B dengan manipulasi URL, API, atau cookie.

### Fase 6 — Tanya jawab, rekap dan CSV

**Tugas:** Go API sesi presentasi, catatan partisipasi, rekap dan CSV hanya guru; frontend form rekap/filter/unduh.
**Done jika:** guru dapat memantau kewajiban pertanyaan per sesi dan mengambil laporan CSV aman.

### Fase 7 — Tes menyeluruh, deploy single-VPS, dokumentasi

**Tugas:** `go test ./...`, unit/service, HTTP `httptest`, integration test PostgreSQL, Playwright E2E browser, dan tes keamanan; dokumentasi backup/restore; deploy ke `cbt-vps` dengan pola LMS: build frontend static + backend di VPS, migrasi Goose, backup binary, atomic swap, restart systemd, health retry, dan rollback otomatis bila `/readyz` gagal. Konfigurasikan HTTPS, cookie, environment, log terstruktur, serta smoke route frontend.
**Done jika:** AC01–AC25 lulus; satu binary Go melayani API dan static frontend, semua mutasi ditolak tanpa admin, deployed Git HEAD/migration/systemd/`healthz`/`readyz` terverifikasi, dan tidak ada integrasi Gmail.

### Standar tiap fase

1. Jalankan `go test ./...`, `go vet ./...`, dan build Go pada fase backend.
2. Jalankan typecheck/lint/build Next.js untuk fase frontend.
3. Verifikasi migrasi pada database uji; pastikan data seed dummy bisa dibuat ulang.
4. Tulis ringkasan perubahan, hasil tes, blocker, dan langkah berikutnya di laporan implementasi.

---

## 14. Acceptance criteria (wajib diuji)

- [ ] **AC01:** Guru menambah kelas dan tujuh kelompok; semua tampil di publik sebagai `Belum Mengumpulkan`.
- [ ] **AC02:** Hanya PPT dicatat → `Sebagian`.
- [ ] **AC03:** PDF menyusul → `Lengkap`.
- [ ] **AC04:** Data publik tidak memuat email kelompok atau catatan evaluasi rinci.
- [ ] **AC05:** Catatan draf `PPT — Slide 5` tidak terlihat siswa.
- [ ] **AC06:** Setelah guru memublikasikan, kelompok bersangkutan melihat lokasi dan instruksi revisi.
- [ ] **AC07:** Kelompok berbeda tidak dapat melihat catatan tersebut dengan mengganti ID/URL.
- [ ] **AC08:** Revisi PPT putaran 1 tidak menghapus penerimaan PDF lama.
- [ ] **AC09:** `DISETUJUI` ditolak bila ada temuan wajib terbuka atau file digital belum lengkap.
- [ ] **AC10:** `TUNTAS` hanya terjadi bila PPT/PDF lengkap, review disetujui, dan buku diterima.
- [ ] **AC11:** Semua aksi penting memiliki audit log.
- [ ] **AC12:** Request tanpa admin tidak dapat mengubah status atau publikasi, termasuk via API langsung.
- [ ] **AC13:** Rate limit kode, rotasi kode, dan pencabutan sesi berfungsi.
- [ ] **AC14:** Guru dapat mencatat pertanyaan kelompok A pada presentasi kelompok B.
- [ ] **AC15:** Hanya admin dapat ekspor CSV.
- [ ] **AC16:** Tidak ada akses Gmail API/IMAP/OAuth inbox, pengiriman balasan, atau fitur upload file.
- [ ] **AC17:** Tanggal tampil WIB; waktu disimpan UTC.
- [ ] **AC18:** Monitoring, tabel, dan form bisa digunakan pada layar HP.
- [ ] **AC19:** Refresh halaman tidak mengubah data sendiri.
- [ ] **AC20:** Kelas dan tahun ajaran bisa ditambah tanpa mengubah kode sumber.
- [ ] **AC21:** Semua CRUD/status/evaluasi/otorisasi dan ekspor CSV dilayani **API Go**, bukan Next.js Server Actions.
- [ ] **AC22:** Frontend Next.js tidak memiliki direct DB client/SQL dan tidak mengetahui `DATABASE_URL`.
- [ ] **AC23:** `go test ./...`, `go vet ./...`, build backend, serta pengujian endpoint Go berhasil.
- [ ] **AC24:** Login guru, hash password Argon2id, sesi server-side, rotasi/pencabutan token kelompok, cookie dan CSRF berfungsi.
- [ ] **AC25:** Deployment single-VPS di `cbt-vps` berjalan aman: binary Go melayani API + static export Next.js, PostgreSQL memakai role aplikasi terpisah, systemd aktif, HTTPS tersedia, backup/restore terdokumentasi, serta Git HEAD, migrasi, `/healthz`, `/readyz`, dan frontend smoke terverifikasi.

## 15. Panduan operasional guru

**Pra-pengumpulan:** daftarkan kelas/kelompok/topik/email, bagikan kode kelompok secara privat, bagikan juknis.
**Ketika email masuk:** buka Gmail secara manual, cocokkan pengirim dan lampiran, lalu **Catat Pengumpulan** pada kelompok terkait.
**Ketika buku diserahkan:** tandai buku tulis sebagai diterima secara manual.
**Ketika ada kesalahan:** tambah temuan dengan lokasi slide/halaman dan instruksi; simpan draf; publikasi saat final.
**Ketika revisi diterima:** buka Gmail manual, buat putaran revisi baru, tandai temuan yang selesai, perbarui status.
**Setelah selesai:** ekspor CSV dan arsipkan/backup sesuai kebijakan sekolah.

**Peringatan untuk siswa di website:** `Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah. Lihat waktu pembaruan terakhir.`

---

## 16. Risiko dan mitigasi

| Risiko | Mitigasi |
|---|---|
| Email sudah dikirim, dashboard belum berubah | Tampilkan waktu pembaruan; jelaskan guru memperbarui manual; daftar tugas pending di admin |
| Siswa salah mengira sudah terkumpul berarti diterima | Tampilkan status kiriman, evaluasi, dan buku dalam kolom terpisah |
| Kode kelompok bocor | Kode kuat, rotasi, revoke sesi, rate limiting |
| Temuan belum lengkap terlanjur dilihat siswa | Sistem draf dan tombol publikasi eksplisit |
| Revisi menimpa bukti awal | Putaran kiriman immutable secara historis; perubahan tercatat dalam audit |
| Akun email kelompok dibagi tidak aman | Ikuti kebijakan sekolah, jangan simpan password atau membagikannya pada website |
| Data hilang/salah ubah | Backup, audit, konfirmasi tindakan berisiko dan soft delete |

---

## 17. Definition of Done

MVP dianggap selesai ketika: semua fitur wajib sudah berjalan; AC01–AC25 lulus; dokumentasi `README.md`, `DEPLOYMENT.md`, `ADMIN_GUIDE.md`, dan `.env.example` tersedia; guru mampu mencatat kiriman, menulis/publikasi evaluasi, mencatat revisi, dan membaca rekap tanpa SQL atau coding; aplikasi memakai HTTPS dan tidak membocorkan secret/data kelompok; **tidak ada integrasi Gmail**.

## 18. Instruksi untuk AI coding agent

1. **Baca seluruh file PLAN.md sebelum coding.** Perlakukan bagian “Non-negotiable” sebagai persyaratan tetap.
2. Eksekusi fase **berurutan**, mulai dari setup, migrasi/database, autentikasi, lalu UI.
3. Setelah tiap fase, jalankan lint, typecheck, test, dan build yang relevan; perbaiki kegagalan sebelum lanjut.
4. Gunakan data dummy saat pengembangan; jangan mengarang data pengumpulan siswa sesungguhnya.
5. Implementasikan keamanan dan otorisasi **di server**, bukan hanya dengan menyembunyikan komponen UI.
6. Pertahankan histori versi file dan log; jangan mengganti riwayat lama ketika revisi baru dicatat.
7. Jangan menambah fitur upload PPT/PDF ke website, Gmail API, auto-grade AI, atau notifikasi otomatis tanpa permintaan guru.
8. Setelah semua fase, uji skenario AC01–AC25 secara nyata; sertakan laporan tes dan instruksi menjalankan aplikasi.
9. **Backend WAJIB Go**. Seluruh endpoint REST, autentikasi, RBAC, transisi status, transaksi SQL, histori dan audit berada di `backend/`; frontend hanya memanggil API.
10. Gunakan `chi`, `pgxpool`, `sqlc`, dan PostgreSQL; pilih versi stabil saat implementasi. Pastikan semua operasi mutasi lulus tes otorisasi dan keamanan.
11. Gunakan akun dan data dummy untuk testing. Jangan pernah meminta akun Gmail kelompok atau guru dari aplikasi.

### Prompt siap pakai untuk coding agent

> Bangun MVP website monitoring dan evaluasi tugas kelompok Informatika kelas IX berdasarkan PLAN.md ini. Implementasikan secara bertahap dari Fase 0 sampai Fase 7 dengan **backend Go (chi, pgxpool, sqlc, PostgreSQL)** dan **frontend Next.js + TypeScript + Tailwind CSS**. Seluruh API bisnis, autentikasi, status, evaluasi, dan audit wajib di Go; Next.js hanya mengurus UI dan proxy. Seluruh penerimaan email, perubahan status, pemeriksaan, dan revisi dilakukan manual oleh guru. Tidak ada integrasi Gmail atau upload file ke website. Utamakan keamanan data antar kelompok, catatan revisi spesifik per lokasi, histori, dan pengujian acceptance criteria. Hasilkan kode Go dan Next.js yang benar-benar berjalan, migrasi SQL, seed, Dockerfile backend, dokumen OpenAPI, dokumentasi setup/deploy, dan hasil tes AC01–AC25.

---

## 19. Keputusan default yang dapat disesuaikan nanti

- Ringkasan status publik; **detail koreksi privat per kelompok** menggunakan kode akses.
- Guru boleh mengisi tenggat revisi, tetapi tidak ada notifikasi otomatis.
- Nilai akhir/rubrik numerik belum dihitung otomatis karena bobot belum dikunci dalam spesifikasi web.
- Sistem mendukung banyak kelas dan tahun ajaran.
- Juknis dapat ditautkan di website jika guru menyediakan URL yang sah.

> **Kalimat kunci untuk pengembang:** “Email hanya tempat pengumpulan file. Website hanya tempat guru mencatat status dan memublikasikan evaluasi, semuanya manual. Backend wajib Go; Next.js hanya frontend.”
