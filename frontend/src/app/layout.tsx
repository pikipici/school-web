import type { Metadata } from "next";
import "@fontsource/newsreader/700.css";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-sans/700.css";
import "@fontsource/ibm-plex-mono/500.css";
import "./globals.css";

export const metadata: Metadata = {
  title: "Papan Progres Informatika",
  description:
    "Monitoring publik pengumpulan, pemeriksaan, dan buku tulis tugas kelompok Informatika kelas IX.",
  applicationName: "Papan Progres Informatika"
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="id">
      <body>
        <a className="skip-link" href="#konten-utama">
          Lewati ke konten utama
        </a>
        {children}
      </body>
    </html>
  );
}
