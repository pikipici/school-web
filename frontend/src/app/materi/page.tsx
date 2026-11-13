import type { Metadata } from "next";
import { MaterialList } from "@/components/public/ContentViews";import { SiteFooter,SiteHeader } from "@/components/site/SiteChrome";
export const metadata:Metadata={title:"Materi"};export default function Page(){return <><SiteHeader/><main id="konten-utama" tabIndex={-1}><h1 className="page-title">Materi</h1><p className="section-note">Bahan belajar Informatika yang disusun menurut topik.</p><MaterialList/></main><SiteFooter/></>}
