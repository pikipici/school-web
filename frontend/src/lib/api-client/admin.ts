// Typed client for the Go admin authentication API. It only talks to the
// same-origin backend; sessions live in an HttpOnly cookie the browser manages,
// so this module never sees or stores the session token.

export const ADMIN_LOGIN_PATH = "/api/v1/admin/auth/login" as const;
export const ADMIN_LOGOUT_PATH = "/api/v1/admin/auth/logout" as const;
export const ADMIN_ME_PATH = "/api/v1/admin/auth/me" as const;

const DEFAULT_RETRY_SECONDS = 60;
const MAX_RETRY_SECONDS = 3600;

export interface AdminSession {
  username: string;
  /** Send as X-CSRF-Token on every unsafe admin request. Keep in memory only. */
  csrfToken: string;
  /** Absolute session limit, ISO 8601 (UTC). */
  expiresAt: string;
}

export interface LoginCredentials {
  username: string;
  password: string;
}

export type AdminApiErrorKind =
  | "invalid_credentials"
  | "rate_limited"
  | "busy"
  | "forbidden"
  | "invalid_request"
  | "network"
  | "server";

const MESSAGES: Record<AdminApiErrorKind, string> = {
  invalid_credentials: "Nama pengguna atau kata sandi salah. Periksa kembali, lalu coba lagi.",
  rate_limited: "Terlalu banyak percobaan masuk. Tunggu sebentar sebelum mencoba lagi.",
  busy: "Server sedang sibuk. Tunggu beberapa detik, lalu coba lagi.",
  forbidden: "Permintaan ditolak oleh server. Muat ulang halaman, lalu coba lagi.",
  invalid_request: "Data yang dikirim tidak valid. Periksa isian Anda, lalu coba lagi.",
  network: "Tidak dapat terhubung ke server. Periksa koneksi Anda, lalu coba lagi.",
  server: "Terjadi gangguan pada server. Coba lagi beberapa saat lagi."
};

/** Fixed, user-safe text for an error kind. Server-provided text is never shown. */
export function messageFor(kind: AdminApiErrorKind): string {
  return MESSAGES[kind];
}

export class AdminApiError extends Error {
  kind: AdminApiErrorKind;
  retryAfterSeconds?: number;

  constructor(kind: AdminApiErrorKind, retryAfterSeconds?: number) {
    super(messageFor(kind));
    this.name = "AdminApiError";
    this.kind = kind;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

export interface AdminApiDeps {
  fetch?: typeof fetch;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const isNonEmptyString = (value: unknown): value is string => typeof value === "string" && value.trim() !== "";

function isAdminSession(value: unknown): value is AdminSession {
  return (
    isRecord(value) &&
    isNonEmptyString(value.username) &&
    isNonEmptyString(value.csrfToken) &&
    isNonEmptyString(value.expiresAt) &&
    !Number.isNaN(Date.parse(value.expiresAt))
  );
}

const isAbort = (error: unknown): boolean => error instanceof DOMException && error.name === "AbortError";

async function send(url: string, init: RequestInit, deps: AdminApiDeps): Promise<Response> {
  const doFetch = deps.fetch ?? fetch;
  try {
    return await doFetch(url, { ...init, credentials: "same-origin", cache: "no-store" });
  } catch (error) {
    if (isAbort(error)) {
      throw error;
    }
    throw new AdminApiError("network");
  }
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

function parseRetryAfter(header: string | null): number {
  if (header !== null && /^\d+$/.test(header)) {
    const seconds = Number(header);
    if (seconds >= 1 && seconds <= MAX_RETRY_SECONDS) {
      return seconds;
    }
  }
  return DEFAULT_RETRY_SECONDS;
}

const CODE_TO_KIND: Record<string, AdminApiErrorKind> = {
  INVALID_CREDENTIALS: "invalid_credentials",
  RATE_LIMITED: "rate_limited",
  SERVICE_BUSY: "busy",
  FORBIDDEN: "forbidden",
  INVALID_REQUEST: "invalid_request",
  PAYLOAD_TOO_LARGE: "invalid_request",
  UNSUPPORTED_MEDIA_TYPE: "invalid_request"
};

async function toError(response: Response): Promise<AdminApiError> {
  const body = await readJson(response);
  const code = isRecord(body) && isRecord(body.error) && typeof body.error.code === "string" ? body.error.code : "";
  const kind: AdminApiErrorKind = CODE_TO_KIND[code] ?? (response.status === 429 ? "rate_limited" : "server");
  return new AdminApiError(kind, kind === "rate_limited" ? parseRetryAfter(response.headers.get("Retry-After")) : undefined);
}

async function readSession(response: Response): Promise<AdminSession> {
  const body = await readJson(response);
  if (isRecord(body) && isAdminSession(body.data)) {
    const { username, csrfToken, expiresAt } = body.data;
    return { username, csrfToken, expiresAt };
  }
  throw new AdminApiError("server");
}

/** Sign in. Resolves with the session; rejects with AdminApiError. */
export async function login(credentials: LoginCredentials, deps: AdminApiDeps = {}): Promise<AdminSession> {
  const response = await send(
    ADMIN_LOGIN_PATH,
    {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ username: credentials.username, password: credentials.password })
    },
    deps
  );
  if (!response.ok) {
    throw await toError(response);
  }
  return readSession(response);
}

/** Current session, or null when signed out. Other failures reject. */
export async function getSession(signal?: AbortSignal, deps: AdminApiDeps = {}): Promise<AdminSession | null> {
  const response = await send(ADMIN_ME_PATH, { method: "GET", headers: { Accept: "application/json" }, signal }, deps);
  if (response.status === 401) {
    return null;
  }
  if (!response.ok) {
    throw await toError(response);
  }
  return readSession(response);
}

/** Sign out. Rejects if the server did not end the session. */
export async function logout(csrfToken: string, deps: AdminApiDeps = {}): Promise<void> {
  const response = await send(
    ADMIN_LOGOUT_PATH,
    { method: "POST", headers: { Accept: "application/json", "X-CSRF-Token": csrfToken } },
    deps
  );
  if (!response.ok) {
    throw await toError(response);
  }
}
