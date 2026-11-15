import type { Metadata } from "next";
import { Suspense } from "react";

import { AdminNav } from "@/components/admin/AdminPageGate";
import { GroupDetailPage } from "@/components/admin/GroupDetail";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Pengumpulan kelompok",
  robots: { index: false, follow: false }
};

export default function Page() {
  return (
    <>
      <SiteHeader />
      <main id="konten-utama" tabIndex={-1}>
        <AdminNav current="kelompok" />
        <Suspense fallback={<p role="status">Memuat halaman…</p>}>
          <GroupDetailPage />
        </Suspense>
      </main>
      <SiteFooter />
    </>
  );
}
