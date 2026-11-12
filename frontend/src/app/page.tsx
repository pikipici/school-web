import Link from "next/link";

import { ProjectSummary } from "@/components/public/ProjectSummary";

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

export default function Home() {
  return (
    <>
      <header className="site-header" aria-label="Kepala halaman">
        <div className="header-inner">
          <Link className="brand" href="/" aria-label="Papan Progres Informatika, beranda">
            <span className="brand-mark" aria-hidden="true">P</span>
            <span>Papan Progres Informatika</span>
          </Link>
          <span className="school-level">Kelas IX</span>
        </div>
      </header>

      <main id="konten-utama" tabIndex={-1}>
        <section className="intro" aria-labelledby="judul-halaman">
          <p className="eyebrow">Monitoring tugas kelompok</p>
          <h1 id="judul-halaman">Progres kelas yang jelas dan dapat dipercaya.</h1>
          <p className="lede">
            Papan ini membantu siswa melihat tahap pengumpulan, pemeriksaan, dan penyerahan buku tulis tanpa membuka data pribadi kelompok.
          </p>
        </section>

        <aside className="manual-notice" aria-labelledby="peringatan-manual">
          <span className="notice-icon" aria-hidden="true">!</span>
          <div>
            <h2 id="peringatan-manual">Pembaruan dilakukan manual</h2>
            <p>Status diperbarui manual oleh guru. Jika email baru saja dikirim, status mungkin belum berubah.</p>
          </div>
        </aside>

        <section className="progress-section" aria-labelledby="alur-progres">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Tiga jalur terpisah</p>
              <h2 id="alur-progres">Cara membaca progres</h2>
            </div>
            <p>Setiap jalur menjelaskan tahap yang berbeda dan tidak digabung menjadi satu status umum.</p>
          </div>

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

      <footer>
        <p>Papan informasi publik · Informatika kelas IX</p>
        <p>Tidak terhubung ke Gmail dan tidak memeriksa tugas secara otomatis.</p>
      </footer>
    </>
  );
}
