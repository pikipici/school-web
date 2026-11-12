import type { Metadata } from "next";

import { AdminShell } from "@/components/admin/AdminShell";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Dasbor Admin",
  robots: { index: false, follow: false }
};

export default function AdminPage() {
  return (
    <>
      <SiteHeader />

      <main id="konten-utama" tabIndex={-1}>
        <h1 className="page-title">Dasbor Admin</h1>
        <AdminShell />
      </main>

      <SiteFooter />
    </>
  );
}
