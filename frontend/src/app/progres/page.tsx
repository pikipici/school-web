import type { Metadata } from "next";

import { ProjectSummary } from "@/components/public/ProjectSummary";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Progres Tugas Kelompok",
  description: "Pengumpulan, pemeriksaan, dan buku tulis tugas kelompok Informatika kelas IX."
};

const lanes = [
  {
    number: "01",
    title: "Pengumpulan",
    description: "PPT dan PDF dicatat terpisah setelah guru memeriksa kiriman email."
  },
  {
    number: "02",
    title: "Pemeriksaan",
    description: "Hasil pemeriksaan dan kebutuhan revisi diterbitkan secara sengaja oleh guru."
  },
  {
    number: "03",
    title: "Buku Tulis",
    description: "Penyerahan buku fisik kelompok dicatat sebagai jalur progres tersendiri."
  }
] as const;

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
            <p>Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah.</p>
          </div>
        </aside>

        <section className="content-section" aria-labelledby="alur-progres">
          <h2 id="alur-progres">Tiga jalur progres</h2>
          <p className="section-note">Setiap jalur menjelaskan tahap yang berbeda dan tidak digabung menjadi satu status umum.</p>

          <ol className="lane-list">
            {lanes.map((lane) => (
              <li className="lane" key={lane.number}>
                <span className="lane-number" aria-hidden="true">{lane.number}</span>
                <div>
                  <h3>{lane.title}</h3>
                  <p>{lane.description}</p>
                </div>
                <span className="lane-state">
                  <span aria-hidden="true">○</span> Menunggu data kelompok
                </span>
              </li>
            ))}
          </ol>

          <ProjectSummary />
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
