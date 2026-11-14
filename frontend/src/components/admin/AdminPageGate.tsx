"use client";

import Link from "next/link";
import type { ReactNode } from "react";

import { useAdminSession } from "./useAdminSession";

/**
 * Shows loading/error states while the admin session is checked, then renders
 * children with the CSRF token. Cosmetic only: Go enforces access.
 */
export function AdminPageGate({ children }: { children: (csrf: string, onExpired: () => void) => ReactNode }) {
  const { state, retry, expired } = useAdminSession();
  if (state.status === "loading" || state.status === "redirecting") {
    return (
      <p className="public-state loading" role="status">
        {state.status === "loading" ? "Memeriksa sesi masuk…" : "Mengalihkan ke halaman masuk…"}
      </p>
    );
  }
  if (state.status === "error") {
    return (
      <div className="public-state error" role="alert">
        <p>Sesi belum dapat diperiksa. Periksa koneksi Anda, lalu coba lagi.</p>
        <button className="retry-button" type="button" onClick={retry}>
          Coba lagi
        </button>
      </div>
    );
  }
  return <>{children(state.session.csrfToken, expired)}</>;
}

/** Links between admin areas. */
export function AdminNav({ current }: { current: "dasbor" | "kelas" | "kelompok" }) {
  const items = [
    { key: "dasbor", href: "/admin/", label: "Dasbor (Pengumuman dan Materi)" },
    { key: "kelas", href: "/admin/kelas/", label: "Kelas dan kelompok" }
  ] as const;
  return (
    <nav aria-label="Menu admin">
      <ul className="admin-nav">
        {items.map((i) => (
          <li key={i.key}>
            <Link href={i.href} aria-current={current === i.key || (current === "kelompok" && i.key === "kelas") ? "page" : undefined}>
              {i.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
