# Fase 0 — Acceptance Evidence

> Tanggal: 13 November 2026. Deployed HEAD di `cbt-vps`: `e779d36` (commit setelahnya hanya dokumen).
>
> **Verdict: DITUTUP DENGAN PENGECUALIAN (instruksi user, 13 Nov 2026).** Semua gate otomatis dan kontrak PASS. Cek browser nyata belum dilakukan dan dicatat sebagai risiko terbuka, bukan PASS.

## Block A — Hard gate

| Gate | Hasil | Bukti |
|---|---|---|
| Tanpa Gmail/upload/AI grading/DB langsung dari frontend | PASS | Pencarian kode `frontend/src` dan `backend`: satu-satunya kemunculan "Gmail" adalah teks footer yang menyatakan tidak terhubung. Tidak ada Route Handler, Server Action, Prisma, atau Supabase. |
| Tanpa data/statistik/testimoni palsu | PASS | Ringkasan publik `totalGroups: 0`; UI menampilkan keadaan kosong jujur. |
| Setiap link/tombol berfungsi | PASS | Hanya link brand ke `/` dan tombol "Coba lagi" (memanggil ulang API). Skip-link menuju `#konten-utama`. |
| HTML semantik, `lang=id`, hierarki heading | PASS (statik) | `<html lang="id">`, satu `<main>`, h1 → h2 → h3, `<footer>`, viewport meta. |
| Kontras (dihitung) | PASS dengan catatan | Lihat tabel kontras. |
| Reduced motion | PASS | `prefers-reduced-motion` ada di CSS; tidak ada animasi lain. |
| Fokus keyboard | PASS (statik) | `:focus-visible` outline 3px kontras 4.88:1; tombol minimal 44px. Belum diuji dengan keyboard nyata. |
| 360px, menengah, desktop, zoom 200% | **BELUM DIVERIFIKASI** | Breakpoint CSS ada (720px, 390px) dan layout reflow secara kode. Tidak ada browser yang bisa menjangkau `cbt-vps` (port 8300 hanya internal; tool browser menolak alamat lokal). Wajib dicek manual. |

### Tabel kontras (WCAG, dihitung)

| Pasangan | Rasio | Syarat | Hasil |
|---|---|---|---|
| Teks / canvas | 14.62 | 4.5 | PASS |
| Teks redup / canvas | 7.37 | 4.5 | PASS |
| Teks redup / surface | 8.27 | 4.5 | PASS |
| Teks / surface-muted (lane-state) | 13.10 | 4.5 | PASS |
| Primary / canvas | 6.90 | 4.5 | PASS |
| Warning / warning-soft | 5.71 | 4.5 | PASS |
| Danger / danger-soft | 6.30 | 4.5 | PASS |
| Success / success-soft | 5.86 | 4.5 | PASS |
| Putih / danger (hover tombol) | 8.18 | 4.5 | PASS |
| Focus ring / canvas | 4.88 | 3.0 | PASS |
| Garis `--line` / canvas | 2.42 | 3.0 | **di bawah 3:1** |

Catatan garis: `--line` hanya memisahkan blok dekoratif; semua informasi tetap lewat teks. Tombol "Coba lagi" memakai border `--danger` (8.03:1). Mode `prefers-contrast: more` menggelapkan garis. Dicatat sebagai temuan rendah, bukan blocker.

## Block B — Purpose gate

PASS (tinjauan dokumen): warna, tipografi, dan status mengikuti token `DESIGN.md`. Tiga jalur (Pengumpulan, Pemeriksaan, Buku Tulis) tampil terpisah. Tidak ada gambar hasil generate, hero dekoratif, atau kartu statistik.

## Block C — Quality locks

| Lock | Hasil |
|---|---|
| Loading, kosong, error (+ coba lagi), sukses | PASS (kode `ProjectSummary.tsx`) |
| Status memakai teks + ikon + warna | PASS |
| Mobile reflow nyata | PASS di CSS; verifikasi visual belum dilakukan (lihat Block A) |

## Block D — Engineering delivery

Dijalankan 13 Nov 2026 di `cbt-vps`: `bash deploy/deploy.sh --check` (tidak mengubah runtime).

| Gate | Hasil |
|---|---|
| `go test ./... -count=1`, `go vet`, build | PASS |
| `sqlc generate`, Goose preflight | PASS |
| `npm ci`, lint, typecheck, `next build` | PASS |
| `npm audit --omit=dev` | PASS, 0 kerentanan |
| `npm audit` (termasuk dev) | **7 moderate** (rantai `@typescript-eslint`, hanya alat dev; tidak masuk bundel produksi). Belum diperbaiki. |
| Service `informatika-monitoring-api` | active, release `20261112T140834Z-e779d364ec24` |
| `/healthz` | `{"status":"ok"}` |
| `/readyz` | `{"status":"ok","database":"ok"}` |
| `/api/v1/public/summary` | zero-state jujur, `manualUpdate: true` |
| API tak dikenal (juga `%2f`) | JSON 404 |
| Rute frontend tak dikenal | HTML 404 |
| Path traversal | 400 |
| Working tree server | bersih |
| Dokumen (README, LOCAL_AI_CONTEXT, DEPLOYMENT, PROGRESS, OpenAPI) | Disinkronkan 13 Nov 2026 |

Catatan tidak kritis: Tailwind memberi peringatan "No utility classes detected" (CSS kustom dipakai); npm menawarkan versi mayor 12 (tidak dipakai).

## Item terbuka

1. **Wajib:** cek manual di browser nyata — 360px, ±768px, desktop, zoom 200%, navigasi keyboard, pembaca layar singkat. Cara: `ssh -L 8300:127.0.0.1:8300 cbt-vps` lalu buka `http://localhost:8300` di browser sendiri.
2. 7 advisory npm moderate (dev-only): putuskan perbaiki atau terima.
3. Garis `--line` 2.42:1: putuskan gelapkan atau terima.
4. Hostname/HTTPS publik: di luar scope Fase 0.
5. Docker Compose opsional hanya ditinjau statis.
6. Persetujuan user untuk masuk Fase 1.

## Keputusan penutupan

User menginstruksikan penutupan Fase 0 pada 13 Nov 2026 setelah redesign web pribadi ter-deploy (commit `8300336`, release `20261112T204643Z-830033683d94`, smoke PASS, `/` dan `/progres/` 200).

Ditutup dengan pengecualian yang tercatat:

- Gate browser nyata (360px, menengah, desktop, zoom 200%, keyboard): **tidak diverifikasi**. Wajib dilakukan sebelum Fase 1 dianggap menyentuh UI lagi.
- 7 advisory npm moderate (dev-only): diterima sementara, ditinjau ulang di Fase 1.
- Garis `--line` sudah digelapkan ke 3.64:1 (lulus 3:1) pada redesign.
- Fase 1 menunggu persetujuan eksplisit user.
