import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export default function Home() {
  return (
    <>
      <SiteHeader />

      <main id="konten-utama" tabIndex={-1}>
        <h1 className="page-title">Beranda</h1>

        <section className="content-section" id="pengumuman" aria-labelledby="judul-pengumuman">
          <h2 id="judul-pengumuman">Pengumuman</h2>
          <div className="empty-block" role="status">
            <h3>Belum ada pengumuman</h3>
            <p>Pengumuman terbaru dari guru akan tampil di sini, lengkap dengan tanggal dalam WIB.</p>
          </div>
        </section>

        <section className="content-section" id="materi" aria-labelledby="judul-materi">
          <h2 id="judul-materi">Materi</h2>
          <div className="empty-block" role="status">
            <h3>Belum ada materi</h3>
            <p>Daftar materi per topik akan tampil di sini. Lampiran PDF dan PPT dari guru akan dapat diunduh.</p>
          </div>
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
