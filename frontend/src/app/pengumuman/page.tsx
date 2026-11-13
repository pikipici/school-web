import type { Metadata } from "next";
import { AnnouncementList } from "@/components/public/ContentViews";import { SiteFooter,SiteHeader } from "@/components/site/SiteChrome";
export const metadata:Metadata={title:"Pengumuman"};export default function Page(){return <><SiteHeader/><main id="konten-utama" tabIndex={-1}><h1 className="page-title">Pengumuman</h1><p className="section-note">Informasi terbaru untuk kegiatan pembelajaran Informatika.</p><AnnouncementList/></main><SiteFooter/></>}
