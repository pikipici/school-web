"use client";

import { useCallback, useEffect, useState } from "react";

import { getPublicSummary } from "@/lib/api-client/public";
import type { PublicSummary } from "@/types/public";

type SummaryState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "success"; summary: PublicSummary };

export function ProjectSummary() {
  const [state, setState] = useState<SummaryState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((current) => current + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();

    getPublicSummary(controller.signal).then(
      (summary) => setState({ status: "success", summary }),
      (error: unknown) => {
        if (!(error instanceof DOMException && error.name === "AbortError")) {
          setState({ status: "error" });
        }
      }
    );

    return () => controller.abort();
  }, [attempt]);

  if (state.status === "loading") {
    return (
      <div className="summary-state summary-loading" role="status" aria-live="polite" aria-busy="true">
        <span className="summary-icon" aria-hidden="true">…</span>
        <div>
          <h3>Memuat ringkasan proyek</h3>
          <p>Mohon tunggu sementara data monitoring diminta.</p>
        </div>
      </div>
    );
  }

  if (state.status === "error") {
    return (
      <div className="summary-state summary-error" role="alert">
        <span className="summary-icon" aria-hidden="true">!</span>
        <div>
          <h3>Ringkasan belum dapat dimuat</h3>
          <p>Periksa koneksi Anda, lalu coba muat kembali ringkasan proyek.</p>
          <button className="retry-button" type="button" onClick={retry}>Coba lagi</button>
        </div>
      </div>
    );
  }

  if (state.summary.totalGroups === 0) {
    return (
      <div className="summary-state summary-empty" role="status" aria-live="polite">
        <span className="summary-icon" aria-hidden="true">—</span>
        <div>
          <h3>Belum ada kelompok yang disiapkan</h3>
          <p>Daftar monitoring akan muncul setelah guru menyiapkan kelas dan kelompok.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="summary-state summary-success" role="status" aria-live="polite">
      <span className="summary-icon" aria-hidden="true">✓</span>
      <div>
        <h3>Ringkasan proyek tersedia</h3>
        <p>
          <strong>{state.summary.totalGroups}</strong> kelompok tercatat dalam monitoring publik.
        </p>
      </div>
    </div>
  );
}
