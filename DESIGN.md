---
version: alpha
name: Papan Progres Informatika
description: Papan kendali kelas yang hangat, jelas, dan dapat dipercaya untuk monitoring tugas kelompok kelas IX.
colors:
  ink: "#182018"
  inkMuted: "#465046"
  canvas: "#F3F0E5"
  surface: "#FFFDF6"
  surfaceMuted: "#E7E4D8"
  line: "#9A9D91"
  primary: "#185C49"
  primaryHover: "#104536"
  primarySoft: "#D9EBE2"
  information: "#1D5D78"
  informationSoft: "#DDECF2"
  warning: "#795000"
  warningSoft: "#F5E6B8"
  danger: "#8F2D24"
  dangerSoft: "#F4DDD8"
  success: "#25623E"
  successSoft: "#DCEBDD"
  focus: "#AA4D00"
typography:
  display:
    fontFamily: Newsreader
    fontSize: 3rem
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: "-0.025em"
  h1:
    fontFamily: Newsreader
    fontSize: 2.25rem
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.02em"
  h2:
    fontFamily: Newsreader
    fontSize: 1.625rem
    fontWeight: 650
    lineHeight: 1.2
    letterSpacing: "-0.01em"
  h3:
    fontFamily: IBM Plex Sans
    fontSize: 1.125rem
    fontWeight: 700
    lineHeight: 1.35
  body:
    fontFamily: IBM Plex Sans
    fontSize: 1rem
    fontWeight: 400
    lineHeight: 1.6
  bodyStrong:
    fontFamily: IBM Plex Sans
    fontSize: 1rem
    fontWeight: 600
    lineHeight: 1.5
  label:
    fontFamily: IBM Plex Sans
    fontSize: 0.875rem
    fontWeight: 600
    lineHeight: 1.35
  data:
    fontFamily: IBM Plex Mono
    fontSize: 0.875rem
    fontWeight: 500
    lineHeight: 1.45
rounded:
  sm: 3px
  md: 7px
  lg: 12px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 40px
  section: 72px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#FFFFFF"
    typography: "{typography.bodyStrong}"
    rounded: "{rounded.sm}"
    padding: 12px
    height: 44px
  button-primary-hover:
    backgroundColor: "{colors.primaryHover}"
    textColor: "#FFFFFF"
    typography: "{typography.bodyStrong}"
    rounded: "{rounded.sm}"
    padding: 12px
    height: 44px
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.bodyStrong}"
    rounded: "{rounded.sm}"
    padding: 12px
    height: 44px
  status-pending:
    backgroundColor: "{colors.surfaceMuted}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: 8px
  status-warning:
    backgroundColor: "{colors.warningSoft}"
    textColor: "{colors.warning}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: 8px
  status-danger:
    backgroundColor: "{colors.dangerSoft}"
    textColor: "{colors.danger}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: 8px
  status-success:
    backgroundColor: "{colors.successSoft}"
    textColor: "{colors.success}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: 8px
---

## Overview

`Papan Progres Informatika` terasa seperti papan kendali kelas yang disusun guru, bukan dashboard SaaS generik. Antarmuka harus hangat, tegas, mudah dipindai siswa, dan efisien saat guru mencatat banyak kelompok. Karakter utamanya datang dari tiga jalur status yang selalu dipisahkan: **Pengumpulan**, **Pemeriksaan**, dan **Buku Tulis**.

Antislop digunakan sebagai filter, bukan sumber gaya. Setiap teknik visual harus melayani hierarki, keterbacaan, status, atau efisiensi tugas. Jika nama sekolah dan nama aplikasi dilepas, komposisi tiga jalur status, tipografi editorial, serta permukaan seperti kertas kerja tetap harus membuat produk ini dikenali.

### Product truth

- Status diperbarui manual oleh guru setelah memeriksa Gmail di luar website.
- Website tidak membaca Gmail, tidak menerima upload berkas, dan tidak memeriksa tugas otomatis.
- Monitoring publik hanya berisi informasi aman dan ringkas.
- Catatan detail hanya terlihat oleh guru dan kelompok pemiliknya setelah dipublikasikan.
- Tidak ada ranking, statistik palsu, testimonial, klaim otomatis, atau copy pemasaran berlebihan.

### Liveliness dials

- **Energy 2/5:** tenang dan fokus pada pekerjaan kelas.
- **Rhythm 4/5:** variasi density yang disengaja antara ringkasan, daftar, dan editor evaluasi.
- **Motion 1/5:** hanya feedback state dan transisi singkat 120–180 ms.

## Colors

Canvas memakai warna kertas hangat, bukan putih steril. Tinta hijau-hitam memberi rasa akademik tanpa terlihat korporat. Hijau mineral adalah warna aksi utama; ochre, merah oksida, dan biru informasi hanya digunakan untuk makna status.

Aturan:

- Jangan memakai gradient dekoratif, glow, glassmorphism, atau background grid.
- Jangan memakai warna sebagai satu-satunya pembeda status; selalu sertakan ikon dan teks.
- Aksen tidak boleh muncul pada setiap kartu, ikon, link, dan garis sekaligus.
- Teks utama memakai `ink`; teks sekunder memakai `inkMuted` hanya pada ukuran normal.
- Focus ring memakai `focus`, terlihat minimal 2px dan berjarak dari batas kontrol.
- Semua pasangan warna wajib dihitung; target WCAG AA 4.5:1 untuk teks normal dan 3:1 untuk komponen/focus indicator.

## Typography

`Newsreader` dipakai terbatas pada judul halaman dan heading bagian agar halaman terasa seperti catatan akademik yang serius. `IBM Plex Sans` dipakai pada isi, navigasi, form, dan kontrol. `IBM Plex Mono` hanya untuk data yang memang teknis: waktu, kode kelompok yang sedang dikelola guru, ID audit, dan nama file metadata.

Aturan:

- Body text minimal 16px pada halaman publik dan portal kelompok.
- Gunakan `clamp()` untuk display/H1 agar tidak memotong layar sempit.
- Jangan memakai monospace sebagai dekorasi atau seluruh heading.
- Jangan memakai uppercase + letter spacing lebar sebagai pola heading standar.
- Satu halaman memiliki satu H1 dan urutan heading tidak boleh dilompati.
- Front-load informasi terpenting pada heading dan paragraf pertama.

## Layout

### Public monitoring

- Desktop: header ringkas, penjelasan manual-update, filter, kemudian daftar kelompok.
- Kolom status tetap terpisah: Kelompok, Kelas, Topik, Pengumpulan, Pemeriksaan, Buku Tulis, Terakhir Diperbarui.
- Ringkasan angka hanya ditampilkan jika membantu siswa/guru memahami progres nyata; bukan baris kartu KPI dekoratif.
- Pada mobile, tabel berubah menjadi record rows satu kolom dengan label field; jangan memaksa horizontal scroll.

### Group portal

- Status kelompok sendiri muncul pertama.
- Tiga jalur status disusun sebagai urutan kerja, bukan tiga kartu identik.
- Catatan revisi dikelompokkan berdasarkan putaran dan file.
- Setiap temuan menampilkan lokasi, masalah, tindakan yang diminta, prioritas, dan status penyelesaian.

### Admin

- Desktop: navigasi kiri + workspace utama. Area tindakan yang perlu ditangani didahulukan daripada statistik.
- Mobile: navigasi utama tetap mudah ditemukan; menu sekunder menggunakan tombol berlabel `Menu`, bukan ikon hamburger tanpa nama.
- Form panjang memakai satu kolom pada mobile. Tombol utama berada dekat akhir form dan tidak ditutup fixed navigation.
- Editor evaluasi menjaga konteks kelompok, putaran, dan file tetap terlihat.

### Responsive states

Layout harus dirancang pada minimal tiga keadaan berdasarkan titik pecah konten:

1. Narrow: satu kolom, record rows, action bar yang membungkus.
2. Intermediate: dua kolom bila konten benar-benar muat; sidebar boleh menjadi drawer berlabel.
3. Wide: sidebar dan workspace penuh.

Tidak boleh ada page-level horizontal overflow pada lebar 360px. Semua target sentuh minimal 44×44px dengan jarak yang memadai. Konten harus tetap terbaca pada zoom 200%.

## Elevation & Depth

Depth berasal dari layering permukaan dan garis, bukan shadow lembut pada semua benda.

- Surface utama memakai border 1px `line`.
- Shadow hanya untuk dialog, popover, atau elemen yang benar-benar berada di atas layer lain.
- Sticky header/navigation harus ringkas dan tidak menutupi focus target.
- Jangan menjadikan setiap bagian sebagai floating card.

## Shapes

- Radius kecil menunjukkan alat kerja yang presisi.
- Tombol, input, status, tabel, dan dialog tidak boleh semuanya berbentuk pill.
- Pill hanya untuk nilai pendek yang memang kategorikal, dan tetap harus punya teks bermakna.
- Ikon tidak dimasukkan ke kotak bulat dekoratif tanpa fungsi.

## Components

### Status indicator

Status selalu memiliki:

1. ikon sederhana yang relevan,
2. label Bahasa Indonesia lengkap,
3. warna semantik,
4. optional supporting text untuk timestamp atau penyebab.

Contoh label yang benar: `Belum Mengumpulkan`, `Terkumpul Sebagian`, `Sedang Diperiksa`, `Perlu Revisi`, `Buku Sudah Diterima`. Hindari hanya `Pending`, `Done`, atau dot berwarna.

### Buttons and links

- Gunakan kata kerja spesifik: `Catat Pengumpulan`, `Tambah Temuan`, `Publikasikan Evaluasi`, `Unduh CSV`.
- Hindari CTA generik seperti `Mulai`, `Explore`, `Learn More`, atau panah dekoratif.
- Semua button harus bekerja; aksi yang belum tersedia tidak dirender.
- Link navigasi memakai `<a href>`; aksi memakai `<button type="button">` atau submit yang tepat.
- Semua kontrol enabled memakai pointer cursor dan memiliki hover, active, disabled, focus-visible state.

### Forms

- Setiap input memiliki visible `<label>`; placeholder bukan label.
- Required/optional ditulis eksplisit.
- Error menjelaskan masalah dan perbaikan, bukan hanya border merah.
- Publikasi evaluasi dan rotasi kode meminta preview + konfirmasi.
- Draft dan Published dibedakan dengan teks/ikon, bukan warna saja.
- Focus input pada mobile tidak boleh tertutup keyboard.

### Data views

Semua daftar/tabel/editor data memiliki:

- loading skeleton yang menyerupai struktur konten dan label teks loading,
- empty state yang menjelaskan arti keadaan kosong dan tindakan valid,
- error state yang menjelaskan kegagalan dan opsi retry,
- stale/manual-update notice bila status mungkin belum mengikuti email terbaru.

### Dialogs

- Judul menyebut tindakan dan objek.
- Escape menutup dialog bila aman.
- Focus ditahan di dalam dialog dan kembali ke pemicu saat ditutup.
- Destructive/irreversible action tidak menjadi default focus tanpa alasan.

### Tables

- Header jelas dan sticky hanya jika benar-benar membantu daftar panjang.
- Sorting/filtering memiliki label yang terbaca screen reader.
- Mobile memakai semantic record list, bukan tabel desktop yang dipotong.
- Data privat tidak pernah dimasukkan ke DOM halaman publik lalu hanya disembunyikan dengan CSS.

## Do's and Don'ts

### Do

- Tampilkan peringatan: `Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah.`
- Pisahkan status pengumpulan, pemeriksaan, dan buku tulis pada semua konteks.
- Prioritaskan daftar tindakan guru: belum dicatat, perlu diperiksa, perlu dipublikasikan, dan perlu revisi.
- Gunakan copy Bahasa Indonesia yang langsung dan spesifik.
- Tampilkan waktu dalam WIB sambil menyimpan waktu UTC pada backend.
- Gunakan semantic HTML, skip link, landmarks, keyboard navigation, dan visible focus.
- Hormati `prefers-reduced-motion` dan `prefers-contrast` bila relevan.
- Uji halaman 360px, intermediate/tablet, desktop, keyboard-only, dan zoom 200%.

### Don't

- Jangan membuat hero marketing besar, trusted-logo bar, testimonial, pricing, FAQ generik, atau tiga feature cards.
- Jangan memakai gradient ungu-biru, glow crypto, glassmorphism, bento grid, fake terminal, sparkle, atau AI capsule badge.
- Jangan mengarang angka progres, uptime, jumlah sekolah, nilai, atau statistik siswa.
- Jangan menyembunyikan navigasi desktop tanpa alasan.
- Jangan membuat tombol yang tidak memiliki implementasi.
- Jangan menampilkan email kelompok, nama file, draft evaluasi, audit, token, atau catatan internal pada monitoring publik.
- Jangan menggunakan generated logo/illustration/photo tanpa persetujuan user terpisah.
- Jangan menambahkan dependency ikon/animasi hanya untuk dekorasi; gunakan CSS dan simbol sederhana bila cukup.
