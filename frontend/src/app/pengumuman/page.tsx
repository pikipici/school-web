import type { Metadata } from "next";
import { AnnouncementList } from "@/components/public/ContentViews";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = { title: "Pengumuman" };
export default function Page() {
 return <><SiteHeader /><main id="konten-utama" className="public-library" tabIndex={-1}>
  <div className="library-intro"><p className="page-eyebrow">KABAR KELAS</p><h1 className="page-title">Pengumuman</h1></div>
  <AnnouncementList />
 </main><SiteFooter /></>;
}
