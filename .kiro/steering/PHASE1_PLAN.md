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
| 1.3 | Paket sandi (Argon2id, kebijakan sandi) | TDD: hash cocok, salah ditolak, sandi lemah ditolak |
| 1.4 | Paket sesi (buat, cek idle dan absolut, hapus) | TDD dengan jam palsu |
| 1.5 | Batas percobaan login | TDD: kunci, pulih, per IP dan per akun |
| 1.6 | Handler login/logout/me + middleware auth dan CSRF | TDD HTTP: 401, 403 CSRF, cookie flag |
| 1.7 | CLI `adminctl` (buat, reset sandi) | Uji; sandi dari prompt atau stdin, tidak lewat argumen |
| 1.8 | Audit log pada login/logout | Uji: tidak ada rahasia di log |
| 1.9 | Frontend `/admin/login` dan `/admin` | lint, typecheck, build; keadaan loading/error/sukses |
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
