"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import { AdminApiError, getSession, login, messageFor } from "@/lib/api-client/admin";

const ADMIN_HOME = "/admin/";

export function LoginForm() {
  const router = useRouter();
  const passwordRef = useRef<HTMLInputElement>(null);

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<AdminApiError | null>(null);
  const [blockedUntil, setBlockedUntil] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());

  // Someone who is already signed in goes straight to the dashboard.
  useEffect(() => {
    const controller = new AbortController();
    getSession(controller.signal).then(
      (session) => {
        if (session) {
          router.replace(ADMIN_HOME);
        }
      },
      () => undefined
    );
    return () => controller.abort();
  }, [router]);

  // Tick once a second while a rate-limit wait is running.
  useEffect(() => {
    if (blockedUntil === null) {
      return;
    }
    const timer = window.setInterval(() => {
      const current = Date.now();
      setNow(current);
      if (current >= blockedUntil) {
        setBlockedUntil(null);
      }
    }, 1000);
    return () => window.clearInterval(timer);
  }, [blockedUntil]);

  const secondsLeft = blockedUntil === null ? 0 : Math.max(0, Math.ceil((blockedUntil - now) / 1000));
  const waiting = secondsLeft > 0;

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting || waiting) {
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      await login({ username: username.trim(), password });
      setPassword("");
      router.replace(ADMIN_HOME);
    } catch (caught) {
      const apiError = caught instanceof AdminApiError ? caught : new AdminApiError("server");
      setError(apiError);
      setPassword("");
      if (apiError.kind === "rate_limited") {
        const start = Date.now();
        setNow(start);
        setBlockedUntil(start + (apiError.retryAfterSeconds ?? 60) * 1000);
      }
      setSubmitting(false);
      passwordRef.current?.focus();
    }
  }

  const canSubmit = username.trim() !== "" && password !== "" && !submitting && !waiting;

  return (
    <form className="form-card" onSubmit={onSubmit} noValidate aria-busy={submitting}>
      {error ? (
        <div className="form-error" role="alert">
          <span className="notice-icon" aria-hidden="true">!</span>
          <p>
            {messageFor(error.kind)}
            {waiting ? ` Coba lagi dalam ${secondsLeft} detik.` : ""}
          </p>
        </div>
      ) : null}

      <div className="form-field">
        <label htmlFor="username">Nama pengguna</label>
        <input
          id="username"
          name="username"
          type="text"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          maxLength={64}
          required
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />
      </div>

      <div className="form-field">
        <label htmlFor="password">Kata sandi</label>
        <input
          id="password"
          name="password"
          type="password"
          autoComplete="current-password"
          maxLength={128}
          required
          ref={passwordRef}
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </div>

      <button className="button-primary" type="submit" disabled={!canSubmit}>
        {submitting ? "Memeriksa…" : waiting ? `Tunggu ${secondsLeft} detik` : "Masuk ke dasbor"}
      </button>
    </form>
  );
}
