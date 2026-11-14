import type { Metadata } from "next";

import { AdminNav } from "@/components/admin/AdminPageGate";
import { ClassManagerPage } from "@/components/admin/ClassManager";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Kelas dan kelompok",
  robots: { index: false, follow: false }
};

export default function Page() {
  return (
    <>
      <SiteHeader />
      <main id="konten-utama" tabIndex={-1}>
        <h1 className="page-title">Kelas dan kelompok</h1>
        <AdminNav current="kelas" />
        <ClassManagerPage />
      </main>
      <SiteFooter />
    </>
  );
}
