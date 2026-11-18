# Panduan Guru — Memberi Tahu Kelompok Letak Kesalahan

Website: [fikriramaaulia.online](https://fikriramaaulia.online). Siswa tidak perlu akun. Hanya guru yang login.

## Yang dilihat siswa

| Terlihat publik | Hanya guru |
|---|---|
| Nama kelompok, kelas, topik | Email kelompok |
| Status pengumpulan, pemeriksaan, buku tulis | Pengirim, subjek, nama berkas |
| Catatan perbaikan yang sudah **diterbitkan** | Catatan perbaikan yang masih **draf** |
| | Catatan guru di putaran dan buku tulis |

Jangan menulis nama siswa atau data pribadi di catatan perbaikan. Catatan yang sudah terbit bisa dibaca siapa saja dan tidak bisa diubah.

## 1. Masuk

Buka `/admin/login/`, isi **Nama pengguna** dan **Kata sandi**. Sesi berakhir setelah 30 menit tanpa aktivitas.

## 2. Siapkan kelas dan kelompok (sekali di awal)

1. Menu **Kelas dan kelompok** (`/admin/kelas/`).
2. Bagian **Tambah kelas**: isi kode (mis. `9A`), lalu **Tambah kelas**.
3. Di baris kelas, klik **Kelompok**.
4. Bagian **Tambah kelompok**: nama, topik, email kelompok (opsional), lalu simpan.

Kelas atau kelompok yang sudah punya catatan tidak bisa dihapus biasa. Pilihannya:

- **Nonaktifkan**: hilang dari halaman publik, data tetap ada, bisa diaktifkan lagi.
- **Hapus permanen** (khusus kelompok): klik **Hapus**, lalu ketik nama kelompok persis dan klik **Hapus permanen**. Semua kiriman, buku tulis, dan catatan perbaikan kelompok itu ikut terhapus, termasuk yang sudah terbit. Tidak bisa dibatalkan; hanya bisa kembali lewat cadangan database.

## 3. Catat kiriman dari Gmail

Website tidak membaca Gmail. Periksa Gmail sendiri, lalu:

1. Di daftar kelompok, klik **Pengumpulan** pada kelompok itu.
2. Isi **Email diterima (WIB)**, centang **PPT** dan/atau **PDF**, lalu **Simpan putaran**.
3. Revisi berikutnya dicatat sebagai putaran baru. PDF lama tetap terhitung walau revisi hanya berisi PPT.
4. Buku tulis fisik: bagian **Buku tulis**, centang **Buku tulis sudah diterima**, lalu **Simpan buku tulis**.

## 4. Tulis letak kesalahan

Di halaman kelompok yang sama, bagian **Catatan perbaikan**:

1. Pilih **Jenis tugas** (PPT, PDF, BUKU, PRESENTASI).
2. **Letak kesalahan**, mis. `Slide 5` atau `Halaman 3, paragraf 2`.
3. **Masalah**, mis. `Gambar stack terbalik`.
4. **Perbaikan yang diminta**, mis. `Balik arah panah: elemen terakhir masuk ada di atas`.
5. **Prioritas**: **Wajib diperbaiki** atau **Saran**.
6. **Simpan draf**. Draf belum terlihat siswa. Ulangi untuk setiap kesalahan.

## 5. Terbitkan

1. Periksa semua draf. Setelah terbit, kata-katanya tidak bisa diubah.
2. Simpan atau batalkan formulir yang masih terbuka.
3. Pilih **Status pemeriksaan**. Biasanya **Perlu Revisi** bila ada kesalahan wajib.
4. Klik **Terbitkan catatan dan status**, lalu **Ya, terbitkan ke publik**.

Semua draf kelompok itu terbit bersamaan dengan status yang dipilih.

## 6. Setelah siswa memperbaiki

1. Catat revisi yang masuk (langkah 3).
2. Ubah status ke **Sedang Diperiksa**, lalu **Simpan status**.
3. Pada tiap kesalahan yang sudah benar, klik **Tandai selesai**.
4. Bila semua kesalahan wajib selesai dan PPT + PDF lengkap, pilih **Materi Disetujui**.

Kelompok **Tuntas** bila PPT + PDF lengkap, status Materi Disetujui, dan buku tulis diterima. Tuntas dihitung otomatis.

## 7. Cek dari sisi siswa

Buka [fikriramaaulia.online/progres/](https://fikriramaaulia.online/progres/) di jendela penyamaran (tanpa login). Cari nama kelompok. Catatan perbaikan yang terbit tampil di bawah statusnya.

## Bila ada masalah

| Pesan | Artinya |
|---|---|
| "Sesi Anda telah berakhir" | Login lagi. |
| "Data ini sudah diubah di tempat lain" | Ada tab lain yang mengubah. Halaman dimuat ulang; ulangi. |
| "Hanya putaran terakhir yang bisa dihapus" | Koreksi putaran lama lewat **Ubah**. |
| "Masih punya data terkait" | Kelas: hapus dulu kelompoknya. Kelompok: pakai **Hapus permanen** atau nonaktifkan. |
| "Nama kelompok yang diketik tidak sama" | Ketik ulang persis, termasuk huruf besar/kecil. |
