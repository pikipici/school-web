import type { Metadata } from "next";
import { ProgressTable } from "@/components/public/ProgressTable";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = { title: "Progres Tugas Kelompok", description: "Lihat progres tugas setiap kelompok dan catatan perbaikannya tanpa login." };
export default function ProgressPage() {
 return <><SiteHeader/><main id="konten-utama" className="progress-home" tabIndex={-1}>
  <div className="progress-intro"><p className="page-eyebrow">INFORMATIKA · KELAS IX</p><h1 className="page-title">Progres Tugas Kelompok Kelas 9</h1>
  <p>Cek Penilaian Tugas Kelompok</p></div>
  <section aria-label="Pencarian dan progres kelompok"><ProgressTable/></section>
  <p className="manual-footnote">Status diperbarui manual oleh guru, bukan otomatis dari email. Jika baru mengirim tugas, tunggu pembaruan dan periksa waktu pada kartu kelompok.</p>
 </main><SiteFooter/></>;
}
