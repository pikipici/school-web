# Fase 4 — Tanya Jawab, Rekap, CSV, Audit; Fase 5 — Penutupan MVP (Rencana)

> Status: **USULAN 18 Nov 2026 — menunggu persetujuan user.** Dikerjakan setelah Fase 3 (3C, 3D) selesai.
> Sumber: `PLAN.md` bagian 2.4, 6E, 7 (rute `/admin/partisipasi`, `/admin/rekap`, `/admin/audit`), 9 (`presentation_sessions`, `question_participation`), 10 (endpoint), 13 (Fase 6–7 lama), 14 (AC14, AC15, AC01–AC25), 15–16 (kriteria MVP selesai).

## Tujuan

- **Fase 4:** guru mencatat sesi presentasi dan pertanyaan antarkelompok, melihat rekap semua kelompok, mengunduh CSV, dan membaca log audit dari dasbor.
- **Fase 5:** membuktikan MVP selesai: AC01–AC25 diuji nyata, panduan guru, latihan pulih cadangan, uji keamanan menyeluruh.

## Pembagian

| Bagian | Isi | Fase lama `PLAN.md` | Risiko utama |
|---|---|---|---|
| **4A** | Sesi presentasi + pertanyaan antarkelompok | 6 | Kelompok penanya = penyaji, atau beda kelas |
| **4B** | Rekap, ekspor CSV, halaman audit | 6 (+ rute audit bagian 7) | CSV bocor ke non-admin; injeksi rumus di Excel |
| **5** | Penutupan MVP | 7 | Klaim "lulus" tanpa bukti nyata |

## Keputusan yang saya usulkan

| # | Hal | Usulan | Alasan |
|---|---|---|---|
| P1 | Sesi presentasi | Satu baris per presentasi: kelas, kelompok penyaji, tanggal tampil (WIB di form), catatan guru opsional. Satu kelompok boleh tampil lebih dari sekali (mis. tampil ulang) | `PLAN.md` 9; kenyataan kelas |
| P2 | Partisipasi | Satu baris per (sesi, kelompok penanya): `siap` dan `diajukan` (centang), catatan guru opsional. Unik per sesi | `PLAN.md` 2.4, 6E |
| P3 | Teks pertanyaan | **Opsional**, maks 500 karakter, hanya guru | Pertanyaan ada di buku kelompok; teks tidak wajib disalin |
| P4 | Aturan kelas | Penanya ≠ penyaji, dan harus sekelas; dijaga di Go **dan** trigger DB | AC14; `PLAN.md` 9 |
| P5 | Kewajiban terpenuhi | Dihitung: kelompok memenuhi sesi itu bila `diajukan`; `siap` saja = "Siap, belum bertanya". Tidak disimpan | Sama dengan status 3B: dihitung, bukan diisi tangan |
| P6 | Form | Halaman sesi menampilkan **semua kelompok lain sekelas** dengan dua centang; satu tombol **Simpan**; kirim massal dalam satu transaksi + audit | Guru mengisi cepat saat sesi berjalan |
| P7 | Publik dan portal | Partisipasi **tidak** tampil di publik. Portal kelompok (3D) **tidak** diubah di Fase 4 | Data minimum; bisa ditambah nanti |
| P8 | Rekap | `/admin/rekap/`: satu baris per kelompok aktif: kelas, nama, topik, Pengumpulan, Pemeriksaan, Buku, Tuntas, jumlah putaran, temuan wajib terbuka, sesi dipenuhi "n dari m". Filter kelas dan status | `PLAN.md` 7; menggantikan cek satu per satu |
| P9 | Isi CSV | Kolom sama dengan rekap + waktu terakhir diperbarui (WIB). **Tanpa** email, catatan internal, teks temuan, atau teks pertanyaan | Berkas mudah tersebar; isi rinci tetap di dasbor |
| P10 | Format CSV | UTF-8 dengan BOM (Excel membaca huruf beraksen), pemisah koma, sel yang diawali `= + - @`, tab, atau CR diberi awalan `'` | Cegah injeksi rumus |
| P11 | Akses CSV | `GET /api/v1/admin/reports/export.csv` hanya dengan sesi admin; `Cache-Control: no-store`; `Content-Disposition: attachment`; tercatat audit `report.exported` (filter saja) | AC15 |
| P12 | Halaman audit | `/admin/audit/`: baca saja, terbaru dulu, berhalaman 50, filter jenis aksi dan tanggal; menampilkan waktu WIB, aksi, hasil, IP, detail aman | `PLAN.md` 7; detail sudah tanpa rahasia sejak Fase 1 |
| P13 | Uji browser otomatis (Fase 5) | Pakai skrip Edge headless di PC (seperti cek 360px 3B) terhadap halaman produksi dengan API tiruan, ditambah alur nyata lewat klien Node di VPS. **Tidak** memasang Playwright di VPS | RAM VPS ±2 GB; tidak ada peramban di VPS |
| P14 | Data uji AC | AC01–AC25 dijalankan di produksi dengan data `UJI` yang dihapus sesudahnya, seperti 3A/3B | Bukti nyata, produksi kembali kosong |

## Endpoint (garis besar)

| Method | Path | Siapa |
|---|---|---|
| GET/POST | `/api/v1/admin/presentations` | admin (`?class=`) |
| GET/PATCH/DELETE | `/api/v1/admin/presentations/{id}` | admin |
| GET/PUT | `/api/v1/admin/presentations/{id}/participation` | admin: baca/simpan semua baris sekaligus |
| GET | `/api/v1/admin/reports/summary` | admin: data rekap (`?class=&status=`) |
| GET | `/api/v1/admin/reports/export.csv` | admin: unduh CSV |
| GET | `/api/v1/admin/audit` | admin: log (`?action=&from=&to=&page=`) |

Mutasi: sesi + CSRF + Origin + audit dalam transaksi. `PATCH` memakai `updatedAt` (409 saat basi). Hapus sesi presentasi ikut menghapus baris partisipasinya, tercatat audit.

## Task

### 4A — Tanya jawab

| Task | Isi | Uji |
|---|---|---|
| 4.1 | Migrasi `presentation_sessions`, `question_participation` | uji skema: unik (sesi, penanya), penanya ≠ penyaji, sekelas (trigger), down/up |
| 4.2 | Paket + API: sesi CRUD, simpan partisipasi massal, kewajiban terhitung, audit dalam transaksi | TDD: AC14, penanya beda kelas 422, konflik 409, tanpa sesi 401, tanpa CSRF 403 |
| 4.3 | UI `/admin/partisipasi/` (daftar sesi per kelas) dan `/admin/partisipasi/sesi/?id=` (centang per kelompok) | uji klien; lint/typecheck/build; E2E |
| 4.4 | Deploy + cek browser + `docs/F4A_ACCEPTANCE.md` | semua gerbang |

### 4B — Rekap, CSV, audit

| Task | Isi | Uji |
|---|---|---|
| 4.5 | API rekap, CSV, daftar audit | TDD: AC15 (anonim 401), kolom CSV persis tanpa email/catatan, injeksi rumus diberi awalan, BOM, audit ekspor |
| 4.6 | UI `/admin/rekap/` (tabel, filter, tombol unduh) dan `/admin/audit/` | uji klien; lint/typecheck/build; E2E; 360px |
| 4.7 | Deploy + cek browser + `docs/F4B_ACCEPTANCE.md` | semua gerbang; CSV dibuka di Excel |

### 5 — Penutupan MVP

| Task | Isi | Uji |
|---|---|---|
| 5.1 | `docs/ADMIN_GUIDE.md` (guru tanpa SQL: kelas, kelompok, catat kiriman, evaluasi, kode akses, tanya jawab, rekap, cadangan); latihan pulih cadangan DB + lampiran ke database uji | pulih terbukti; panduan dibaca user |
| 5.2 | Uji keamanan menyeluruh `PLAN.md` 12 (manipulasi ID/URL/cookie, tebak kode, draf lewat API, CSRF asing, CSV tanpa admin, rahasia di respons/log) + uji tampilan otomatis 360px/zoom 200%/keyboard di semua halaman | skrip berulang; semua lulus |
| 5.3 | AC01–AC25 nyata di produksi + `docs/MVP_ACCEPTANCE.md`; perbarui README, DEPLOYMENT, `.env.example`; tutup pengecualian lama yang bisa ditutup | semua AC dengan bukti; user menyetujui MVP selesai |

## Jumlah task tersisa

| Bagian | Task |
|---|---|
| 3C (3.10–3.13) | 4 |
| 3D (3.14–3.17) | 4 |
| 4A (4.1–4.4) | 4 |
| 4B (4.5–4.7) | 3 |
| 5 (5.1–5.3) | 3 |
| **Total** | **18** |

## Risiko dan mitigasi

1. **Penanya salah kelas atau sama dengan penyaji.** Trigger DB + validasi Go + uji mutan.
2. **CSV tersebar.** Tanpa email, catatan, dan teks; hanya admin; tercatat audit.
3. **Injeksi rumus di Excel.** Awalan `'` untuk sel berbahaya; diuji per karakter pemicu.
4. **Rekap lambat.** Satu query agregat per permintaan; data kecil (±28 kelompok); diuji dengan 4 × 7.
5. **Klaim AC tanpa bukti.** Tiap AC di `MVP_ACCEPTANCE.md` menyebut uji, bukti produksi, atau pengecualian jujur.

## Pertanyaan untuk user

1. Setuju teks pertanyaan **opsional** (P3), atau wajib disalin guru?
2. Setuju CSV **tanpa email kelompok** (P9)?
3. Setuju partisipasi **tidak tampil** di portal kelompok (P7)?
