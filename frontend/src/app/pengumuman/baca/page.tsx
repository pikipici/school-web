import type { Metadata } from "next";import { Suspense } from "react";
import { Detail } from "@/components/public/ContentViews";import { SiteFooter,SiteHeader } from "@/components/site/SiteChrome";
export const metadata:Metadata={title:"Baca pengumuman"};export default function Page(){return <><SiteHeader/><main id="konten-utama" tabIndex={-1}><Suspense fallback={<p role="status">Memuat konten…</p>}><Detail type="announcement"/></Suspense></main><SiteFooter/></>}
