import Link from "next/link";

import { SITE_OWNER } from "@/lib/site";
import { PublicNavigation } from "./PublicNavigation";

export function SiteHeader() {
  return (
    <header className="site-header">
      <div className="header-inner">
        <Link className="brand" href="/" aria-label={`${SITE_OWNER}, beranda`}>
          {SITE_OWNER}
        </Link>
        <PublicNavigation />
      </div>
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer>
      <p>{SITE_OWNER} · Informatika kelas IX</p>
      <p>Tidak terhubung ke Gmail dan tidak memeriksa tugas secara otomatis.</p>
    </footer>
  );
}
