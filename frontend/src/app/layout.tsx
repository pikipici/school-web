import type { Metadata } from "next";
import "@fontsource/newsreader/700.css";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-sans/700.css";
import "@fontsource/ibm-plex-mono/500.css";
import "./globals.css";
import { SITE_DESCRIPTION, SITE_OWNER, SITE_TITLE } from "@/lib/site";

export const metadata: Metadata = {
  title: { default: SITE_TITLE, template: `%s — ${SITE_OWNER}` },
  description: SITE_DESCRIPTION,
  applicationName: SITE_OWNER
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
