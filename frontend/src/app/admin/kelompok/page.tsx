import type { Metadata } from "next";
import { Suspense } from "react";

import { AdminNav } from "@/components/admin/AdminPageGate";
import { GroupManagerPage } from "@/components/admin/GroupManager";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Kelompok",
  robots: { index: false, follow: false }
};

export default function Page() {
  return (
    <>
      <SiteHeader />
      <main id="konten-utama" tabIndex={-1}>
        <AdminNav current="kelompok" />
        <Suspense fallback={<p role="status">Memuat halaman…</p>}>
          <GroupManagerPage />
        </Suspense>
      </main>
      <SiteFooter />
    </>
  );
}
