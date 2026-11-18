import type { Metadata } from "next";

import { ProgressTable } from "@/components/public/ProgressTable";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Progres Tugas Kelompok",
  description: "Pengumpulan, pemeriksaan, dan buku tulis tugas kelompok Informatika kelas IX."
};

export default function ProgressPage() {
  return (
    <>
      <SiteHeader />

      <main id="konten-utama" tabIndex={-1}>
        <h1 className="page-title">Progres Tugas Kelompok</h1>

        <aside className="manual-notice" aria-labelledby="peringatan-manual">
          <span className="notice-icon" aria-hidden="true">!</span>
          <div>
            <h2 id="peringatan-manual">Pembaruan dilakukan manual</h2>
            <p>
              Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah. Lihat waktu pembaruan
              terakhir.
            </p>
          </div>
        </aside>

        <section className="content-section" aria-label="Daftar progres kelompok">
          <p className="section-note">
            Tiga jalur dicatat terpisah: Pengumpulan (PPT dan PDF lewat email), Pemeriksaan oleh guru, dan Buku Tulis fisik. Kelompok
            Tuntas bila ketiganya selesai. Catatan perbaikan yang sudah diterbitkan tampil di bawah setiap kelompok dan dapat dibaca tanpa login.
          </p>
          <ProgressTable />
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
