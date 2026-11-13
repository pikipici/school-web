# Fase 1 — Acceptance Evidence

> Tanggal: 13 November 2026. Kode yang dideploy: `d2e3fca` (rilis `20261112T232527Z-d2e3fcac4923`). Commit sesudahnya hanya dokumen.
>
> **Verdict: BELUM DITUTUP — 1 gerbang wajib terbuka (cek browser nyata).** Semua gerbang otomatis dan kontrak lulus. HTTPS sudah aktif sejak 13 Nov 2026 (lihat bagian HTTPS).

## Gerbang penutupan (dari `.kiro/steering/PHASE1_PLAN.md`)

| Gerbang | Hasil | Bukti |
|---|---|---|
| Migrasi naik dan turun bersih; `sqlc generate` hijau | PASS | 5 uji skema (`down-to 0`, lalu `up`); sqlc hijau di setiap run |
| Semua uji Go lulus (unit + integrasi), vet dan build hijau | PASS | Gerbang deploy pada `d2e3fca`: `go test ./...` dengan `REQUIRE_TEST_DB=1` (11 paket, tanpa SKIP), vet, build `api` dan `adminctl`; juga lulus 2 kali dan dengan `-race` saat pengembangan |
| Login, logout, me; 401 dan 403 benar; flag cookie benar | PASS (HTTP, bukan browser) | 17 uji handler; uji end-to-end dengan biner asli dan database uji; produksi: `/me` anonim 401, salah sandi 401 |
| Batas percobaan terbukti oleh uji | PASS | 15 uji `ratelimit` + uji handler 429 dengan `Retry-After` dan pemulihan |
| Tidak ada sandi atau token di log, respons, atau repo | PASS dengan satu catatan | Tabel audit diperiksa uji: tanpa sandi/hash/token. Repo: tidak ada berkas `.env`/dump/kunci terlacak; DSN yang ada hanya fixture uji dan placeholder `dev-only-app-change-me` di `*.env.example`. **Catatan:** log deploy produksi dihapus sebelum satu baris yang cocok kata "password" diperiksa (hampir pasti nama paket `auth/password`); tidak terverifikasi |
| OpenAPI diperbarui | PASS | `docs/openapi.yaml` 0.2.0; YAML valid, semua `$ref` terselesaikan |
| Frontend lint, typecheck, build lulus | PASS | Gerbang deploy pada `d2e3fca`; 33 uji klien API |
| **Cek browser nyata (360px, zoom 200%, keyboard)** | **BELUM** | Tidak ada browser yang dapat menjangkau server dari sesi ini; belum pernah ada yang login lewat browser. Lihat daftar periksa di bawah |
| Deploy ke `cbt-vps` + smoke lulus; cadangan sebelum migrasi terbukti ada | PASS | Smoke PASS; `backups/pre-20261112T232527Z-d2e3fcac4923.dump` (2664 byte, 0600); migrasi versi 3 |
| Tidak ada CRUD bisnis yang bocor ke Fase 1 | PASS | Produksi: `/api/v1/admin/{announcements,materials,groups}` dan `/api/v1/{announcements,materials}` semuanya 404; rute admin hanya `auth/login`, `auth/logout`, `auth/me` |
| User menyetujui pindah ke Fase 2 | BELUM | Menunggu |

## Keadaan produksi (hanya baca, 13 Nov 2026)

- Layanan aktif pada rilis `20261112T232527Z-d2e3fcac4923`; `/readyz` ok.
- Tabel: `admin_users`, `admin_sessions`, `audit_log`, `goose_db_version` (versi 3).
- Satu admin `fikri` ada (hash tersimpan), belum pernah login; tidak ada sesi aktif.
- Audit: `admin.created` sukses (via CLI).
- Dua kata sandi contoh yang tertulis di repo dicoba terhadap produksi dan keduanya ditolak (401), jadi admin tidak memakai sandi yang publik. Hasilnya tambah tiga baris `auth.login` gagal dan tiga hitungan gagal untuk IP 127.0.0.1 (batas 5 per 15 menit): bila login lewat tunnel mendadak ditolak (429), tunggu 15 menit.

## Risiko terbuka

1. **HTTPS aktif (13 Nov 2026), dengan tiga hal yang perlu dijaga:** sertifikat Let's Encrypt (berlaku sampai 2027-02-11, TLS 1.3) diperpanjang otomatis oleh Caddy, yang mensyaratkan DNS tetap **DNS only (awan abu-abu)** dan port 80/443 tetap terbuka; HSTS baru 1 jam dan perlu dinaikkan bertahap; belum ada Content-Security-Policy (ekspor statis Next.js memakai skrip inline).
2. **Cek browser nyata belum dilakukan** (komponen UI tidak punya uji otomatis; hanya logika klien).
3. Batas login kini memakai alamat klien asli (Caddy menimpa `X-Forwarded-For`), jadi tiga percobaan gagal lama dari 127.0.0.1 tidak lagi memengaruhi login lewat HTTPS.
4. Batas yang diketahui: hitungan batas login hilang saat layanan restart; dua `create-admin` yang benar-benar serentak dengan username berbeda bisa lolos keduanya; akun bisa terkunci hingga 15 menit saat serangan terdistribusi.
5. Tabel `goose_db_version` sudah terbuat lebih awal oleh `--check` versi lama (diperbaiki; tidak berbahaya).
6. 7 advisory npm moderate (alat dev) masih terbuka sejak Fase 0.

## Daftar periksa browser (dikerjakan user)

Buka `https://fikriramaaulia.online/admin/login/` di browser (tunnel SSH tidak lagi dipakai dan tidak akan bekerja untuk login, karena aplikasi kini hanya menerima permintaan admin dari origin `https://`).

1. Lebar 360px (DevTools, mode ponsel): tidak ada gulir horizontal; kolom isian dan tombol terbaca; tombol minimal 44px.
2. Zoom 200%: semua teks terbaca, tidak ada yang terpotong.
3. Hanya keyboard: Tab melewati "Lewati ke konten utama", menu, Nama pengguna, Kata sandi, "Masuk ke dasbor" dengan cincin fokus terlihat; Enter mengirim.
4. Sandi salah: pesan galat muncul, kolom sandi kosong, fokus kembali ke sandi.
5. Sandi benar: pindah ke `/admin/`, terlihat "Masuk sebagai fikri" dan batas sesi dalam WIB.
6. Muat ulang `/admin/`: tetap masuk. Klik "Keluar": kembali ke login; tombol Back tidak menampilkan dasbor.
7. DevTools, Application, Cookies: `__Host-admin_session` bertanda HttpOnly, Secure, SameSite=Strict, Path=/.
8. Pembaca layar singkat (opsional): galat diumumkan.

## Keputusan penutupan

Fase 1 ditutup setelah (a) daftar periksa browser lulus, (b) keputusan HTTPS dicatat (dipasang, atau secara sadar ditunda dengan akses hanya lewat tunnel), dan (c) user menyetujui Fase 2.
