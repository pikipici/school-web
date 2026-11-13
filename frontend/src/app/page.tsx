import { HomeContent } from "@/components/public/ContentViews";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";
export default function Home(){return <><SiteHeader/><main id="konten-utama" tabIndex={-1}><h1 className="page-title">Beranda</h1><HomeContent/></main><SiteFooter/></>}
