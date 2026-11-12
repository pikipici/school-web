# Fase 1 — Database dan Login Admin (Rencana)

> Status: **DRAFT — menunggu approval. Belum ada kode.**
> Dibuat: 13 November 2026.

## Tujuan

Guru bisa login dengan aman ke dashboard admin. Database punya tabel dasar untuk akun, sesi, dan audit. Belum ada CRUD Pengumuman, Materi, atau kelompok (itu Fase 2 dan seterusnya).

## Di dalam scope

- Migrasi Goose pertama: `admin_users`, `admin_sessions`, `audit_log`.
- Query sqlc dan lapisan akses data.
- Login, logout, dan "siapa saya" di Go.
- Perintah CLI untuk membuat dan mereset akun admin.
- Halaman `/admin/login` dan kerangka `/admin` (kosong, jujur).
- Audit log untuk login berhasil, login gagal, logout.
- Cadangan database otomatis sebelum migrasi di skrip deploy.

## Di luar scope

- CRUD Pengumuman, Materi, lampiran, kelas, kelompok.
- Kode akses kelompok dan portal kelompok.
- Pendaftaran admin publik, reset sandi lewat email, OAuth.
- Hostname dan HTTPS publik (lihat keputusan 1).

## Keputusan yang saya usulkan

| Hal | Usulan | Alasan |
|---|---|---|
| Akun | Satu admin (guru), dibuat lewat CLI di server | Tidak ada pendaftaran publik |
| Hash sandi | Argon2id | Standar modern, ada di `golang.org/x/crypto` |
| Sandi | Minimal 12 karakter, tidak ada di daftar sandi umum | Cukup kuat tanpa aturan rumit |
| Sesi | Disimpan di PostgreSQL. Token acak 32 byte; yang disimpan hanya hash SHA-256 | Token bocor dari DB tidak bisa dipakai |
| Masa sesi | Idle 30 menit, maksimal 12 jam | Dapat diubah |
| Cookie | `HttpOnly`, `SameSite=Strict`, `Secure` saat HTTPS | Cegah pencurian lewat skrip |
| CSRF | Token per sesi di header `X-CSRF-Token` + cek `Origin` | Semua aksi ubah data terlindungi |
| Batas percobaan | Per IP dan per akun, jeda naik bertahap | Cegah tebak sandi |
| Pesan gagal | Sama untuk "akun tidak ada" dan "sandi salah" | Cegah tebak nama akun |
| Reset sandi | CLI di server (tidak ada email) | Sesuai prinsip tanpa email otomatis |
| Frontend | Hanya UI. Penjaga halaman di Next.js bersifat kosmetik; yang menegakkan tetap Go | Sesuai aturan backend di Go |

## Endpoint

| Method | Path | Fungsi |
|---|---|---|
| POST | `/api/v1/admin/auth/login` | Masuk, set cookie sesi |
| POST | `/api/v1/admin/auth/logout` | Hapus sesi |
| GET | `/api/v1/admin/auth/me` | Info admin dan token CSRF; 401 jika belum login |

Semua dicatat di `docs/openapi.yaml`. Respons memakai amplop error standar yang sudah ada.

## Skema awal (garis besar)

- `admin_users`: id, username (unik), password_hash, dibuat, diperbarui, login_terakhir.
- `admin_sessions`: id, user_id, token_hash (unik), csrf_token, dibuat, terakhir_aktif, kedaluwarsa, ip, user_agent.
- `audit_log`: id, waktu (UTC), aktor, aksi, hasil, ip, detail aman (tanpa sandi atau token).

Waktu disimpan UTC, ditampilkan WIB.

## Task

| Task | Isi | Uji |
|---|---|---|
| 1.1 **DONE** | Database uji terpisah di `cbt-vps` (`informatika_monitoring_test`) dan pembantu uji integrasi | Uji koneksi lulus |
| 1.2 **DONE** | Migrasi 0001–0003 + sqlc | `goose up/down`, `sqlc generate`, uji skema |
| 1.3 **DONE** | Paket sandi (Argon2id, kebijakan sandi) | TDD: hash cocok, salah ditolak, sandi lemah ditolak |
| 1.4 **DONE** | Paket sesi (buat, cek idle dan absolut, hapus) | TDD dengan jam palsu |
| 1.5 **DONE** | Batas percobaan login | TDD: kunci, pulih, per IP dan per akun |
| 1.6 **DONE** | Handler login/logout/me + middleware auth dan CSRF | TDD HTTP: 401, 403 CSRF, cookie flag |
| 1.7 **DONE** | CLI `adminctl` (buat, reset sandi) | Uji; sandi dari prompt atau stdin, tidak lewat argumen |
| 1.8 **DONE** | Audit log pada login/logout | Uji: tidak ada rahasia di log |
| 1.9 **DONE** | Frontend `/admin/login` dan `/admin` | lint, typecheck, build; keadaan loading/error/sukses |
| 1.10 | Deploy: cadangan `pg_dump` sebelum migrasi, migrasi pertama, smoke baru | `deploy --check`, lalu deploy |
| 1.11 | Gerbang penutupan + `docs/F1_ACCEPTANCE.md` | Semua gate, cek browser nyata |

Setiap task: uji gagal dulu (RED), lalu implementasi, lalu go test/vet/build di VPS, lalu commit.

## Risiko dan mitigasi

1. **Login lewat HTTP polos di internet** — sandi bisa disadap. Mitigasi: jangan buka `/admin` ke publik sebelum HTTPS; sampai itu, akses admin lewat tunnel SSH.
2. **Migrasi maju-saja** — rollback binary tidak membatalkan skema. Mitigasi: `pg_dump` sebelum migrasi (Task 1.10).
3. **Tidak ada Docker di VPS** — uji integrasi memakai PostgreSQL lokal VPS dengan database uji terpisah, tanpa menyentuh data produksi.
4. **Penguncian akun** — penyerang bisa mengunci guru. Mitigasi: kunci per IP lebih dulu; jeda bertahap, bukan kunci permanen.
5. **Lupa sandi satu-satunya admin** — Mitigasi: CLI reset di server.

## Keputusan user (13 Nov 2026)

1. **Domain**: `fikriramaaulia.online`. Cek DNS (hanya baca): domain ini memakai Cloudflare (IP `104.21.15.63`, `172.67.161.194`) dan **belum menunjuk ke VPS** (`43.134.111.212`). Port 80/443 di VPS bebas; Caddy/Nginx belum terpasang; `sudo` tanpa sandi tersedia.
2. **Alamat web**: apex `fikriramaaulia.online` langsung.
3. **Username admin**: `fikri`.
4. **Masa sesi**: idle 30 menit, maksimal 12 jam (disetujui).
5. **2FA (TOTP)**: ditunda ke fase berikutnya.

Catatan: apex saat ini masih lewat Cloudflare dan belum menunjuk ke VPS. Mengubahnya ke VPS akan mengganti apa pun yang tampil di alamat itu sekarang; user yang mengubah DNS (H2).

### Rencana HTTPS (perlu persetujuan sebelum menyentuh VPS atau DNS)

| Langkah | Isi | Siapa |
|---|---|---|
| H1 | Putuskan alamat: apex `fikriramaaulia.online` atau subdomain (mis. `www`, `guru`). Pastikan apex tidak dipakai situs lain | User |
| H2 | Buat rekor DNS A ke `43.134.111.212` di Cloudflare | User |
| H3 | Pilih mode Cloudflare: **DNS saja (awan abu-abu)** agar Caddy mengurus sertifikat sendiri (paling sederhana), atau **proxied (awan oranye)** + SSL Full (strict) + Origin Certificate | User |
| H4 | Buka port 80 dan 443 di firewall/security group penyedia VPS | User |
| H5 | Pasang Caddy, `reverse_proxy 127.0.0.1:8300`, header keamanan (HSTS, nosniff), batas ukuran body | Saya (dengan izin) |
| H6 | Set `COOKIE_SECURE=true` dan uji dari luar | Saya |

Selama H1–H6 belum selesai, `/admin` tidak boleh dibuka ke publik; akses admin lewat tunnel SSH.

## Keputusan yang masih saya butuhkan dari user

Semua keputusan rencana sudah dijawab. Yang tersisa: persetujuan memulai Task 1.1, serta H2–H4 (DNS, mode Cloudflare, firewall) oleh user.



## Gerbang selesai Fase 1

- [ ] Migrasi naik dan turun bersih; `sqlc generate` hijau.
- [ ] Semua uji Go lulus (unit + integrasi), vet dan build hijau.
- [ ] Login, logout, me bekerja; 401 dan 403 benar; cookie flag benar.
- [ ] Batas percobaan terbukti oleh uji.
- [ ] Tidak ada sandi atau token di log, respons, atau repo.
- [ ] OpenAPI diperbarui.
- [ ] Frontend lint, typecheck, build lulus; cek browser nyata (360px, zoom 200%, keyboard).
- [ ] Deploy ke `cbt-vps` + smoke lulus; cadangan sebelum migrasi terbukti ada.
- [ ] Tidak ada CRUD bisnis yang bocor ke Fase 1.
- [ ] User menyetujui pindah ke Fase 2.

## Catatan pelaksanaan

- **Task 1.1 selesai (13 Nov 2026).** RED: 
o non-test Go files. GREEN di cbt-vps: 3 uji lulus (validasi DSN, koneksi, reset skema), go vet dan go build ./... hijau, gofmt bersih. Role informatika_monitoring_test (bukan superuser, tanpa CREATEDB/CREATEROLE) memiliki database uji sendiri; tidak punya akses ke database produksi. Kredensial ada di `/home/ubuntu/.informatika-monitoring-test.env` (mode 0600, di luar repo).
- Pembantu `backend/internal/testdb`: menolak nama database yang tidak berakhiran `_test`; tanpa `TEST_DATABASE_URL` uji dilewati, dan gagal bila `REQUIRE_TEST_DB=1`.
- **Untuk Task 1.10:** gate deploy harus memuat file env uji dan memasang `REQUIRE_TEST_DB=1` agar uji integrasi tidak lolos diam-diam.
- **Task 1.2 selesai (13 Nov 2026).** RED: `goose up` gagal tanpa migrasi (5 uji gagal). GREEN di `cbt-vps`: 5 uji skema lulus (tabel ada, constraint username/hash/kedaluwarsa, cascade sesi, audit log append-only dan tetap ada setelah user dihapus, down-to 0 lalu up). `go test ./...` lulus 3 kali berturut-turut, `go vet`, `go build`, `gofmt`, `sqlc generate` hijau.
- Temuan saat GREEN: `go test ./...` menjalankan paket paralel pada satu database uji sehingga reset skema saling bertabrakan. Diperbaiki dengan advisory lock Postgres di `testdb.Open` yang ditahan selama satu uji.
- Migrasi **belum dijalankan di database produksi**. Deploy berikutnya akan menjalankannya, jadi Task 1.10 (cadangan `pg_dump`) wajib selesai sebelum deploy apa pun.
- Query sqlc untuk user/sesi/audit ditulis bersama task yang memakainya (1.3–1.8), bukan di muka.
- **Task 1.3 selesai (13 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps`: uji hash PHC, salt acak, verify benar/salah/unicode, 9 encoding berbahaya ditolak (memori/iterasi absurd, base64 rusak, algoritma lain), rehash otomatis bila parameter lebih lemah, `DummyVerify` untuk meratakan waktu akun tak dikenal, kebijakan sandi (min 12 karakter, maks 128 byte, tidak memuat username, bukan sandi umum, minimal 5 karakter berbeda). `go test ./...` lulus 2 kali berturut-turut, `go vet`, `go build`, `gofmt`, `sqlc generate` hijau.
- Dependensi baru: `golang.org/x/crypto v0.59.0` (terpin); `x/sync` dan `x/text` naik versi sebagai syarat, `go mod verify` OK. Parameter produksi: 64 MiB, 3 iterasi, 2 lajur.
- **Untuk Task 1.5/1.6:** tiap hash memakai 64 MiB RAM. Batasi hash serentak (mis. semaphore 2–4) di handler login agar banjir login tidak menghabiskan memori VPS.
- **Task 1.4 selesai (13 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps` (database uji nyata, jam palsu): 12 uji lulus. Token 32 byte acak, hanya hash SHA-256 yang disimpan (token mentah tidak ada di baris DB); idle 30 menit bergeser saat aktif; batas mutlak 12 jam tetap berlaku walau aktif; `last_seen_at` ditulis paling cepat tiap 1 menit; revoke idempoten; revoke semua sesi satu user tidak menyentuh user lain; purge menghapus sesi idle/kedaluwarsa; hapus user membatalkan sesinya; token cacat ditolak tanpa query DB. Semua kegagalan memakai satu `ErrInvalid`. `go test ./...` lulus 2 kali, vet/build/gofmt/sqlc hijau.
- Refactor: `testdb` kini punya `Goose` dan `OpenMigrated`; `schema_test` memakainya.
- Query sqlc sesi: `backend/sql/queries/admin_sessions.sql`. Pembersihan berkala (`PurgeExpired`) belum dijadwalkan; dipasang di Task 1.6 atau 1.10.
- **Task 1.5 selesai (13 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps`: 15 uji lulus, juga dengan `go test -race` 3 kali. Aturan: 5 gagal per IP atau 10 gagal per akun dalam 15 menit memicu kunci sementara 1, 2, 4, 8, lalu maksimal 15 menit (bukan permanen). Kunci satu IP tidak memblokir IP lain; serangan terdistribusi hanya mengunci akun yang diserang. Username dinormalisasi (huruf kecil, spasi dibuang) dan akun yang tidak ada dihitung sama. IPv6 dihitung per /64; alamat tak terbaca masuk satu keranjang bersama. Memori dibatasi 10.000 kunci per tabel; kunci yang sedang terkunci tidak digusur banjir kunci sampah. `Gate` membatasi hash serentak. `go test ./...`, vet, build, gofmt, sqlc hijau.
- Batas yang diketahui: penyimpanan di memori, jadi hilang saat restart service (dapat diterima untuk satu proses). Akun guru bisa terkunci sampai 15 menit saat serangan terdistribusi.
- **Untuk Task 1.6:** pasang `Check` sebelum verifikasi, `Gate` di sekitar hash (saran 2 slot), `RecordFailure` saat gagal, `RecordSuccess` saat berhasil, dan balas `429` dengan `Retry-After`. IP klien harus diambil dengan benar (di belakang Caddy: header yang dipercaya hanya dari proxy lokal).
- **Task 1.6 selesai (13 Nov 2026).** RED: paket `users` dan simbol `httpapi` belum ada; tes config gagal (`AdminCookieSecure` undefined). GREEN di `cbt-vps` dengan database uji nyata: 17 uji admin + 1 uji config baru, seluruh suite lulus 2 kali dan dengan `-race`; vet, build, gofmt, sqlc hijau. OpenAPI 0.2.0 memuat tiga endpoint baru (YAML valid, semua `\` terselesaikan).
- Perilaku terbukti: cookie `admin_session` HttpOnly + SameSite=Strict + Path=/ + cookie sesi browser; mode HTTPS memakai `__Host-admin_session` + Secure; kegagalan login identik untuk akun salah/tak ada; 400/413/415/405 untuk permintaan cacat; Origin harus sama dengan host; 429 + `Retry-After` saat dikunci lalu pulih; sukses mereset hitungan; `X-Forwarded-For` hanya dipercaya dari proxy loopback (entri paling kanan); me 401/200; idle 30 menit berlaku; logout wajib CSRF + Origin, idempoten, membersihkan cookie; `Require` menolak POST/PUT/PATCH/DELETE tanpa CSRF atau Origin yang benar; rehash otomatis; 503 + `Retry-After` bila kapasitas hash penuh; peristiwa audit dipancarkan tanpa memuat sandi atau salah ketik username yang bukan format username.
- Konfigurasi baru: `ADMIN_COOKIE_SECURE` (true/false/1/0; nilai lain ditolak agar salah ketik tidak melemahkan keamanan). Rute admin hanya aktif bila `DATABASE_URL` ada. Pembersihan sesi tiap 10 menit di `main`.
- **Task 1.8 tinggal** menyediakan implementasi `AuditRecorder` yang menulis ke tabel `audit_log` (titik panggil dan tes peristiwa sudah ada) lalu memasangnya di `main`.
- **Sebelum HTTPS dipasang (H5):** `ADMIN_COOKIE_SECURE` harus `false` (HTTP), dan Caddy wajib menimpa `X-Forwarded-For` dengan alamat klien sebenarnya; jika Cloudflare mode proxied dipakai, alamat klien harus diambil dari header Cloudflare yang tepercaya (butuh perubahan kecil di `clientIP`).
- **Task 1.7 selesai (13 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps` dengan database uji nyata: 13 uji `adminctl` lulus; seluruh suite lulus 2 kali dan dengan `-race`; vet/build/gofmt/sqlc hijau. Perintah: `create-admin --username NAMA` (hanya satu admin; sisipan atomik `INSERT ... WHERE NOT EXISTS`) dan `reset-password --username NAMA` (ganti hash lalu akhiri semua sesi). Sandi hanya lewat terminal tanpa echo (diminta dua kali) atau `--password-stdin` (maks 1 KiB, hanya akhir baris yang dibuang); flag `--password` ditolak sehingga rahasia tidak lewat argv; keluaran tidak pernah memuat sandi atau hash; kebijakan sandi sama dengan login; username dinormalisasi huruf kecil seperti login. Kode keluar: 0 sukses, 1 gagal operasi, 2 salah pakai/input.
- Binary: `backend/cmd/adminctl`. Dependensi baru: `golang.org/x/term v0.48.0` (terpin, `go mod verify` OK). Uji `go build` biner nyata: `help` tanpa DB berjalan; tanpa `DATABASE_URL` berhenti dengan pesan jelas (exit 1).
- **Untuk Task 1.10:** skrip deploy saat ini hanya membangun `cmd/api`; harus ikut membangun `adminctl` ke direktori rilis (`bin/adminctl`) dan runbook menjelaskan cara pakai: `set -a; . /home/ubuntu/informatika-monitoring/.env; set +a; bin/current/adminctl create-admin --username fikri`. Pembuatan akun admin dilakukan setelah migrasi produksi pertama.
- Batas diketahui: dua `create-admin` yang benar-benar serentak dengan username berbeda bisa lolos keduanya (kecil kemungkinannya; operasi manual oleh satu operator).
- Audit CLI (`admin.created`, `admin.password_reset`) ditambahkan di Task 1.8 bersama penyimpan `audit_log`.
- **Task 1.8 selesai (13 Nov 2026).** RED: `no non-test Go files`. GREEN di `cbt-vps` dengan database uji nyata: 7 uji `audit` + 1 uji persistensi HTTP + 1 uji audit CLI; seluruh suite lulus 2 kali dan dengan `-race`; vet/build/gofmt/sqlc hijau. Uji mutan (`Record` dijadikan no-op) membuat tepat dua uji audit gagal, jadi uji menjaga perilaku sebenarnya.
- `internal/audit.Store.Record` tidak pernah menggagalkan pemanggil (kegagalan hanya dicatat tanpa isi event); tetap menulis walau konteks request sudah dibatalkan; event cacat (aksi/hasil di luar format) dibuang; IP tak valid menjadi NULL tanpa membuang event; kunci detail yang mirip rahasia (password, token, secret, cookie, hash, csrf, authorization, credential, api key) dibuang apa pun yang dikirim pemanggil; maksimal 10 kunci, nilai maksimal 200 byte. Tipe audit di `httpapi` kini alias ke paket `audit`.
- Peristiwa yang tercatat: `auth.login` (sukses/gagal), `auth.login_blocked`, `auth.logout`, `admin.created` (sukses/gagal karena admin sudah ada), `admin.password_reset` (sukses/gagal karena akun tak ditemukan) dengan `via=cli` untuk CLI. Input yang ditolak sebelum mengubah apa pun (sandi lemah, salah pakai) tidak dicatat. Terbukti tidak ada sandi, hash, token sesi, token CSRF, atau salah ketik username di tabel.
- Terpasang di `cmd/api` dan `cmd/adminctl`.
- **Task 1.9 selesai (13 Nov 2026), dengan satu hal belum diverifikasi.** RED: modul `admin.ts` belum ada (uji gagal). GREEN di `cbt-vps`: 33 uji klien API lulus dengan `node --test` (tanpa dependensi baru; Node 26 menjalankan TypeScript langsung; ditambah skrip `npm test` dan `allowImportingTsExtensions`), `npm run typecheck`, `npm run lint`, dan `npm run build` lulus (rute `/admin` dan `/admin/login` terbit statis).
- Klien `src/lib/api-client/admin.ts`: pesan error selalu teks tetap berbahasa Indonesia (teks server tidak pernah ditampilkan), `Retry-After` diparsing dengan batas wajar (default 60 detik), kegagalan jaringan dibedakan dari 401, `getSession` mengembalikan `null` hanya untuk 401 (500 tidak dianggap 'keluar'), abort diteruskan, token CSRF hanya di memori, token sesi tidak pernah terlihat JavaScript (cookie HttpOnly).
- UI: `LoginForm` (label terlihat, `autocomplete` username/current-password, tombol berkata kerja 'Masuk ke dasbor', status memeriksa, galat `role=alert`, hitung mundur saat dibatasi, kata sandi dikosongkan setelah galat, sudah-masuk dialihkan ke dasbor) dan `AdminShell` (memuat, galat + coba lagi, dialihkan ke login bila belum masuk, keluar dengan penanganan gagal, kerangka kosong yang jujur, batas sesi dalam WIB). Halaman `noindex`; tidak ditautkan dari menu publik.
- **Uji nyata tanpa mengubah produksi (database uji):** biner `api` dan `adminctl` asli + hasil build statis: `/admin/` `/admin/login/` 200; `me` 401 sebelum login; salah sandi 401; login 200 dengan cookie `HttpOnly; SameSite=Strict`; `me` 200; logout tanpa CSRF 403, dengan CSRF 204; `me` 401 sesudahnya; tabel audit memuat `admin.created`, `auth.login` gagal+sukses, `auth.logout` tanpa rahasia.
- **Belum diverifikasi:** tampilan nyata di browser (360px, zoom 200%, keyboard, pembaca layar). Tidak ada komponen UI yang diuji otomatis (tidak ada jsdom/Testing Library; hanya logika klien API). Wajib dicek manual di Task 1.11.
- Makefile: ditambah `build-adminctl` dan `test-frontend` ke `make verify`. Skrip deploy belum menjalankan `npm test`; ditambahkan di Task 1.10.
