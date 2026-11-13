# Fase 2A — Acceptance Evidence (Pengumuman dan Materi teks)

> Tanggal: 14 November 2026. Kode di produksi: `12bcf48` (rilis `20261113T220824Z-12bcf48de85e`, migrasi versi 6).
> Perbaikan gaya tautan `e86f851` dideploy 14 Nov 2026 05:42 WIB (rilis `20261113T224209Z-27fe481c50fa`, kode `27fe481`), smoke PASS. Perbaikan lanjutan untuk tautan di isi teks (`f4cb002`) dideploy 05:45 WIB (rilis `20261113T224525Z-f4cb002e17e7`), smoke PASS, aturan garis bawah terverifikasi di CSS produksi.
>
> **Verdict: BELUM DITUTUP — 1 hal terbuka:** tampilan `/admin/` dan formulir pada 360px saat masuk (alat ukur agen tidak punya sesi). Halaman publik dan login sudah terukur di 360px dan 200%; semua perbaikan sudah di produksi.

## Gerbang (dari `.kiro/steering/PHASE2_PLAN.md`)

| Gerbang | Hasil | Bukti |
|---|---|---|
| Migrasi up/down bersih; seed topik benar | PASS | 5 uji skema konten (K1–K7 berurutan, `down-to 3` tanpa menyentuh data Fase 1, `up` ulang tanpa duplikasi); gladi 3→6 pada salinan cadangan produksi di database uji; produksi naik ke versi 6 saat deploy |
| Uji Go (unit + integrasi) lulus, juga `-race`; vet, build, gofmt, sqlc hijau | PASS | `deploy.sh --check` pada `e86f851`: 12 paket Go lulus, 0 FAIL/SKIP, integrasi dengan `REQUIRE_TEST_DB=1`; `go test -race ./...` lulus; gofmt kini gerbang deploy |
| Draf tidak pernah muncul di endpoint publik | PASS | Uji `TestPublicDraftAndMissingDetailsAreBothNotFound`, `TestPublicReadsFilterOrderAndPaginate`; E2E Task 2.6; produksi: draf → API publik kosong, terbit → tampil, tarik → kosong dan detail 404 |
| Semua aksi admin wajib sesi + CSRF + Origin; semua tercatat di audit | PASS | Uji HTTP Task 2.4 (401/403/409/422); E2E: CSRF salah 403, tanpa sesi 401; smoke produksi: `/api/v1/admin/*` anonim 401; audit produksi mencatat 7 aksi uji dari IP user tanpa isi rahasia |
| Frontend lint, typecheck, test, build; tidak ada HTML dari isi yang dirender | PASS | 77 uji frontend, lint, typecheck, build pada `e86f851`; tidak ada `dangerouslySetInnerHTML`/`innerHTML` di `frontend/src`; produksi: isi `<b>tebal</b>` tampil sebagai teks, `https://` jadi tautan |
| OpenAPI diperbarui | PASS | `docs/openapi.yaml` 0.3.0 lolos `openapi-spec-validator` 0.7.1 (15 jalur, 21 operasi) |
| Deploy + smoke lulus | PASS | Rilis `20261113T220824Z-12bcf48de85e`; cadangan `pre-20261113T220824Z-12bcf48de85e.dump`; smoke baru (topik, daftar publik, halaman admin, API admin anonim 401) terbukti gagal pada rilis lama dan lulus pada rilis baru |
| Cek browser nyata: buat, terbitkan, lihat publik, tarik, hapus | PASS (alur) | Edge di produksi dengan CSP `enforce`, 14 Nov 05:16–05:22 WIB: judul kosong ditolak; draf; terbit; publik; tarik; materi tanpa topik tampil "Umum"; hapus dengan konfirmasi; keluar. Data uji dihapus; tabel kosong lagi |
| **Cek 360px, zoom 200%, keyboard** | **PASS kecuali halaman admin yang butuh sesi** | Emulasi perangkat lewat DevTools protocol (Edge headless, produksi): 360px `/`, `/pengumuman/`, `/progres/`, `/admin/login/` tanpa gulir samping; `/materi/` sempat melebar ke 469px karena `select` filter topik; diperbaiki di `a4dfc8a`, dideploy (rilis `20261113T233022Z-e4f640237316`), diukur ulang: 360px pada ketujuh rute; 200% (640 CSS px pada 2x) semua halaman tanpa gulir samping. Keyboard: Tab pertama memunculkan "Lewati ke konten utama" dengan cincin fokus oranye. `/admin/` dan formulir tidak terukur karena butuh sesi |
| User menyetujui lanjut ke 2B | BELUM | Menunggu |

## Temuan saat gerbang

1. **Judul di daftar publik tidak tampak seperti tautan** (warna teks biasa, garis bawah tipis/tak terlihat). Penyebab: reset CSS Tailwind (`a { text-decoration: inherit }`) menghapus garis bawah semua tautan. Diperbaiki di `e86f851` (dideploy): judul daftar, "Kembali ke daftar", dan "Lihat semua …" berwarna utama dengan garis bawah (kontras 6,9:1). Tautan `https://` di dalam isi teks punya masalah yang sama; diperbaiki di `f4cb002` (dideploy).
2. `publishedAt` dari API berzona `+07:00` (zona server), bukan `Z`. Tetap ISO 8601 yang sah; klien memformat ke WIB dengan benar. Tidak ada tindakan.
3. `adminItem` di Go menghilangkan nilai nol (`omitempty`): isi kosong, `pinned` false, `sortOrder` 0, topik kosong. Klien mengisi nilai bawaan dan terbukti di E2E serta produksi. Dicatat di OpenAPI.

## Catatan tampilan kecil (tidak menghambat)

- Di 360px, tombol paginasi "Berikutnya" turun ke baris kedua. Masih bisa dipakai; bisa dirapikan nanti.

## Risiko terbuka

1. Komponen UI tidak punya uji otomatis; tata letak dan keyboard hanya dicek manual.
2. 7 advisory npm moderate (alat dev saja; produksi 0) sejak Fase 0.
3. HSTS 1 tahun menunggu sekitar 27 Nov 2026.
4. Tidak ada pemindai virus; relevan baru di 2B.

## Daftar periksa (dikerjakan user)

Buka [halaman login](https://fikriramaaulia.online/admin/login/) di Edge atau Chrome.

1. **360px:** DevTools (F12) → ikon ponsel → lebar 360. Buka `/`, `/pengumuman/`, `/materi/`, `/admin/`, dan formulir "Tulis pengumuman". Tidak ada gulir ke samping; tombol tidak terpotong.
2. **Zoom 200%:** Ctrl + `+` sampai 200%. Teks terbaca, tidak ada yang tertimpa.
3. **Keyboard saja:** di `/admin/`, tekan Tab terus. Fokus terlihat di setiap tombol. Buka formulir dengan Enter, isi, pilih status dengan panah, simpan dengan Enter. Di baris, "Hapus" lalu fokus pindah ke "Batal"; Esc tidak wajib.
4. Hapus data uji yang dibuat untuk cek ini.

## Keputusan penutupan

Fase 2A ditutup setelah (a) daftar periksa di atas lulus, (b) perbaikan tautan isi teks dideploy dan smoke lulus (selesai), dan (c) user menyetujui lanjut ke 2B.
