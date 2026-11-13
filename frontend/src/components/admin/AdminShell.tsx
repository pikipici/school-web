"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { AdminApiError, getSession, logout, messageFor, type AdminSession } from "@/lib/api-client/admin";

import { ContentManager } from "./ContentManager";

const LOGIN_PATH = "/admin/login/";

type ShellState =
  | { status: "loading" }
  | { status: "redirecting" }
  | { status: "error"; error: AdminApiError }
  | { status: "ready"; session: AdminSession };

const wib = new Intl.DateTimeFormat("id-ID", {
  timeZone: "Asia/Jakarta",
  dateStyle: "long",
  timeStyle: "short"
});

function formatWIB(iso: string): string {
  return `${wib.format(new Date(iso))} WIB`;
}

export function AdminShell() {
  const router = useRouter();
  const [state, setState] = useState<ShellState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const [leaving, setLeaving] = useState(false);
  const [logoutError, setLogoutError] = useState<AdminApiError | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    getSession(controller.signal).then(
      (session) => {
        if (session) {
          setState({ status: "ready", session });
        } else {
          setState({ status: "redirecting" });
          router.replace(LOGIN_PATH);
        }
      },
      (caught: unknown) => {
        if (caught instanceof DOMException && caught.name === "AbortError") {
          return;
        }
        setState({ status: "error", error: caught instanceof AdminApiError ? caught : new AdminApiError("server") });
      }
    );
    return () => controller.abort();
  }, [attempt, router]);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((current) => current + 1);
  }, []);

  const expired = useCallback(() => {
    setState({ status: "redirecting" });
    router.replace(LOGIN_PATH);
  }, [router]);

  async function onLogout(session: AdminSession) {
    setLeaving(true);
    setLogoutError(null);
    try {
      await logout(session.csrfToken);
      router.replace(LOGIN_PATH);
    } catch (caught) {
      setLogoutError(caught instanceof AdminApiError ? caught : new AdminApiError("server"));
      setLeaving(false);
    }
  }

  if (state.status === "loading" || state.status === "redirecting") {
    return (
      <div className="summary-state summary-loading" role="status" aria-live="polite" aria-busy="true">
        <span className="summary-icon" aria-hidden="true">…</span>
        <div>
          <h2>{state.status === "loading" ? "Memeriksa sesi masuk" : "Mengalihkan ke halaman masuk"}</h2>
          <p>Mohon tunggu sebentar.</p>
        </div>
      </div>
    );
  }

  if (state.status === "error") {
    return (
      <div className="summary-state summary-error" role="alert">
        <span className="summary-icon" aria-hidden="true">!</span>
        <div>
          <h2>Sesi belum dapat diperiksa</h2>
          <p>{messageFor(state.error.kind)}</p>
          <button className="retry-button" type="button" onClick={retry}>Coba lagi</button>
        </div>
      </div>
    );
  }

  const { session } = state;
  return (
    <>
      <div className="admin-bar">
        <p className="admin-meta">
          Masuk sebagai <strong>{session.username}</strong>
        </p>
        <button className="button-secondary" type="button" onClick={() => onLogout(session)} disabled={leaving}>
          {leaving ? "Mengakhiri sesi…" : "Keluar"}
        </button>
      </div>

      {logoutError ? (
        <div className="form-error" role="alert">
          <span className="notice-icon" aria-hidden="true">!</span>
          <p>{messageFor(logoutError.kind)} Sesi Anda belum berakhir.</p>
        </div>
      ) : null}

      <ContentManager csrf={session.csrfToken} onExpired={expired} />

      <p className="section-note">
        Sesi berakhir otomatis setelah 30 menit tanpa aktivitas, dan paling lambat pada {formatWIB(session.expiresAt)}.
      </p>
    </>
  );
}
