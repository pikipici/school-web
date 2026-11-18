import type { Metadata } from "next";
import { ProgressTable } from "@/components/public/ProgressTable";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = { title: "Progres Tugas Kelompok", description: "Lihat progres tugas setiap kelompok dan catatan perbaikannya tanpa login." };
export default function Home() {
 return <><SiteHeader/><main id="konten-utama" tabIndex={-1}>
  <h1 className="page-title">Progres Tugas Kelompok</h1>
  <p>Lihat progres kelompokmu dan baca catatan perbaikan tepat di bawahnya. Tidak perlu login.</p>
  <aside className="manual-notice" aria-label="Pembaruan manual"><span className="notice-icon" aria-hidden="true">!</span><p>Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah. Lihat waktu pembaruan terakhir.</p></aside>
  <section className="content-section" aria-labelledby="semua-kelompok"><h2 id="semua-kelompok">Semua kelompok</h2><ProgressTable/></section>
 </main><SiteFooter/></>;
}
