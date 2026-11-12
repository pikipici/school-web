import type { Metadata } from "next";

import { LoginForm } from "@/components/admin/LoginForm";
import { SiteFooter, SiteHeader } from "@/components/site/SiteChrome";

export const metadata: Metadata = {
  title: "Masuk Admin",
  robots: { index: false, follow: false }
};

export default function AdminLoginPage() {
  return (
    <>
      <SiteHeader />

      <main id="konten-utama" tabIndex={-1}>
        <h1 className="page-title">Masuk Admin</h1>
        <p className="section-note">Halaman ini hanya untuk pengelola web. Pengunjung tidak perlu masuk.</p>
        <LoginForm />
      </main>

      <SiteFooter />
    </>
  );
}
