import type { Metadata } from "next";
import { MaterialList } from "@/components/public/ContentViews";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = { title: "Materi" };
export default function Page() {
 return <><SiteHeader /><main id="konten-utama" className="public-library" tabIndex={-1}>
  <div className="library-intro"><p className="page-eyebrow">RUANG BELAJAR</p><h1 className="page-title">Materi Informatika</h1></div>
  <MaterialList />
 </main><SiteFooter /></>;
}
