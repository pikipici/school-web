import type { Metadata } from "next";
import { Suspense } from "react";

import { ContentEditorPage } from "@/components/admin/ContentEditor";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Tulis atau ubah pengumuman",
  robots: { index: false, follow: false }
};

export default function Page() {
  return (
    <>
      <SiteHeader />
      <main id="konten-utama" tabIndex={-1}>
        <Suspense fallback={<p role="status">Memuat formulir…</p>}>
          <ContentEditorPage kind="announcement" />
        </Suspense>
      </main>
      <SiteFooter />
    </>
  );
}