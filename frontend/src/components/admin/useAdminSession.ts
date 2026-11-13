"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { AdminApiError, getSession, type AdminSession } from "@/lib/api-client/admin";

export const LOGIN_PATH = "/admin/login/";

export type SessionState =
  | { status: "loading" }
  | { status: "redirecting" }
  | { status: "error"; error: AdminApiError }
  | { status: "ready"; session: AdminSession };

/**
 * Loads the admin session and sends signed-out visitors to the login page.
 * This guard is only for the interface: the Go server enforces access on
 * every request regardless of what the page shows.
 */
export function useAdminSession() {
  const router = useRouter();
  const [state, setState] = useState<SessionState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

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
        if (caught instanceof DOMException && caught.name === "AbortError") return;
        setState({ status: "error", error: caught instanceof AdminApiError ? caught : new AdminApiError("server") });
      }
    );
    return () => controller.abort();
  }, [attempt, router]);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((n) => n + 1);
  }, []);

  /** Call when the server says the session ended mid-task. */
  const expired = useCallback(() => {
    setState({ status: "redirecting" });
    router.replace(LOGIN_PATH);
  }, [router]);

  return { state, retry, expired };
}
