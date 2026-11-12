import Link from "next/link";

import { NAV_ITEMS, SITE_OWNER } from "@/lib/site";

export function SiteHeader() {
  return (
    <header className="site-header">
      <div className="header-inner">
        <Link className="brand" href="/" aria-label={`${SITE_OWNER}, beranda`}>
          {SITE_OWNER}
        </Link>
        <nav aria-label="Menu utama">
          <ul className="nav-list">
            {NAV_ITEMS.map((item) => (
              <li key={item.href}>
                <Link className="nav-link" href={item.href}>
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
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
