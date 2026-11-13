// Typed client for the Go admin content API (announcements and materials).
// Every write sends the CSRF token from the current session; the session
// cookie itself is HttpOnly and never visible here. Server-provided error text
// is never shown: each failure maps to a fixed Indonesian message.

export type ContentKind = "announcement" | "material";
export type ContentStatus = "draft" | "published";

/** Field limits, enforced the same way by the Go server and the database. */
export const LIMITS = {
  title: 160,
  announcement: { text: 10000 },
  material: { text: 20000 },
  sortOrderMax: 2147483647
} as const;

/** One admin item, with the server's omitted zero values filled in. */
export interface AdminItem {
  kind: ContentKind;
  id: number;
  title: string;
  /** Announcement body or material description. */
  text: string;
  status: ContentStatus;
  pinned: boolean;
  topicId: number | null;
  sortOrder: number;
  publishedAt: string | null;
  createdAt: string;
  /** Exactly as sent by the server; sent back unchanged on update. */
  updatedAt: string;
}

/** The editable part of an item. */
export interface Draft {
  title: string;
  text: string;
  status: ContentStatus;
  pinned: boolean;
  topicId: number | null;
  sortOrder: number;
}

export type DraftErrors = Partial<Record<"title" | "text" | "sortOrder" | "topicId", string>>;

export interface PageMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

export type AdminContentErrorKind =
  | "unauthenticated"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "validation"
  | "too_large"
  | "invalid_request"
  | "network"
  | "server";

const MESSAGES: Record<AdminContentErrorKind, string> = {
  unauthenticated: "Sesi Anda telah berakhir. Silakan masuk kembali.",
  forbidden: "Permintaan ditolak oleh server. Muat ulang halaman, lalu coba lagi.",
  not_found: "Konten tidak ditemukan. Mungkin sudah dihapus.",
  conflict: "Konten ini sudah diubah di tempat lain. Muat ulang data terbaru sebelum menyimpan lagi.",
  validation: "Isian belum sesuai aturan. Periksa judul, isi, dan urutan, lalu coba lagi.",
  too_large: "Isian terlalu panjang untuk disimpan. Persingkat isi, lalu coba lagi.",
  invalid_request: "Permintaan tidak valid. Muat ulang halaman, lalu coba lagi.",
  network: "Tidak dapat terhubung ke server. Periksa koneksi Anda, lalu coba lagi.",
  server: "Terjadi gangguan pada server. Coba lagi beberapa saat lagi."
};

/** Fixed, user-safe text for an error kind. */
export function messageForContent(kind: AdminContentErrorKind): string {
  return MESSAGES[kind];
}

export class AdminContentError extends Error {
  kind: AdminContentErrorKind;

  constructor(kind: AdminContentErrorKind) {
    super(messageForContent(kind));
    this.name = "AdminContentError";
    this.kind = kind;
  }
}

export interface AdminContentDeps {
  fetch?: typeof fetch;
}

const PATHS: Record<ContentKind, string> = {
  announcement: "/api/v1/admin/announcements",
  material: "/api/v1/admin/materials"
};

const CODE_TO_KIND: Record<string, AdminContentErrorKind> = {
  UNAUTHENTICATED: "unauthenticated",
  FORBIDDEN: "forbidden",
  NOT_FOUND: "not_found",
  CONFLICT: "conflict",
  VALIDATION_ERROR: "validation",
  PAYLOAD_TOO_LARGE: "too_large",
  INVALID_REQUEST: "invalid_request",
  UNSUPPORTED_MEDIA_TYPE: "invalid_request"
};

const STATUS_TO_KIND: Record<number, AdminContentErrorKind> = {
  401: "unauthenticated",
  403: "forbidden",
  404: "not_found",
  409: "conflict",
  413: "too_large",
  422: "validation"
};

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const isInt = (v: unknown, min: number): v is number => typeof v === "number" && Number.isSafeInteger(v) && v >= min;
const isDate = (v: unknown): v is string => typeof v === "string" && v.trim() !== "" && !Number.isNaN(Date.parse(v));
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";
const chars = (s: string) => [...s].length;

export function emptyDraft(kind: ContentKind): Draft {
  void kind;
  return { title: "", text: "", status: "draft", pinned: false, topicId: null, sortOrder: 0 };
}

export function draftFrom(item: AdminItem): Draft {
  const { title, text, status, pinned, topicId, sortOrder } = item;
  return { title, text, status, pinned, topicId, sortOrder };
}

/** Client-side check that mirrors the server rules. Empty object = valid. */
export function validateDraft(kind: ContentKind, d: Draft): DraftErrors {
  const errors: DraftErrors = {};
  const title = d.title.trim();
  if (title === "") {
    errors.title = "Judul wajib diisi.";
  } else if (chars(title) > LIMITS.title) {
    errors.title = `Judul paling banyak ${LIMITS.title} karakter.`;
  }
  const max = LIMITS[kind].text;
  if (chars(d.text) > max) {
    errors.text = `Isi paling banyak ${max.toLocaleString("id-ID")} karakter.`;
  }
  if (kind === "material") {
    if (!Number.isInteger(d.sortOrder) || d.sortOrder < 0 || d.sortOrder > LIMITS.sortOrderMax) {
      errors.sortOrder = "Urutan harus bilangan bulat 0 atau lebih.";
    }
    if (d.topicId !== null && !isInt(d.topicId, 1)) {
      errors.topicId = "Pilih topik dari daftar.";
    }
  }
  return errors;
}

function parseItem(kind: ContentKind, v: unknown): AdminItem | null {
  if (!isRecord(v) || !isInt(v.id, 1) || typeof v.title !== "string") return null;
  if (v.status !== "draft" && v.status !== "published") return null;
  if (!isDate(v.createdAt) || !isDate(v.updatedAt)) return null;
  const publishedAt = v.publishedAt === null || v.publishedAt === undefined ? null : v.publishedAt;
  if (publishedAt !== null && !isDate(publishedAt)) return null;
  if (v.status === "published" && publishedAt === null) return null;

  const rawText = kind === "announcement" ? v.body : v.description;
  if (rawText !== undefined && typeof rawText !== "string") return null;
  if (v.pinned !== undefined && typeof v.pinned !== "boolean") return null;
  if (v.sortOrder !== undefined && !isInt(v.sortOrder, 0)) return null;
  if (v.topicId !== undefined && v.topicId !== null && !isInt(v.topicId, 1)) return null;

  return {
    kind,
    id: v.id,
    title: v.title,
    text: rawText ?? "",
    status: v.status,
    pinned: v.pinned ?? false,
    topicId: (v.topicId as number | null | undefined) ?? null,
    sortOrder: (v.sortOrder as number | undefined) ?? 0,
    publishedAt,
    createdAt: v.createdAt,
    updatedAt: v.updatedAt
  };
}

function isMeta(v: unknown): v is PageMeta {
  return isRecord(v) && isInt(v.page, 1) && isInt(v.limit, 1) && isInt(v.total, 0) && isInt(v.totalPages, 0);
}

async function send(url: string, init: RequestInit, deps: AdminContentDeps): Promise<Response> {
  const doFetch = deps.fetch ?? fetch;
  try {
    return await doFetch(url, { ...init, credentials: "same-origin", cache: "no-store" });
  } catch (error) {
    if (isAbort(error)) throw error;
    throw new AdminContentError("network");
  }
}

async function readJson(r: Response): Promise<unknown> {
  try {
    return await r.json();
  } catch {
    return undefined;
  }
}

async function toError(r: Response): Promise<AdminContentError> {
  const body = await readJson(r);
  const code = isRecord(body) && isRecord(body.error) && typeof body.error.code === "string" ? body.error.code : "";
  return new AdminContentError(CODE_TO_KIND[code] ?? STATUS_TO_KIND[r.status] ?? "server");
}

async function readItem(kind: ContentKind, r: Response): Promise<AdminItem> {
  const body = await readJson(r);
  const item = isRecord(body) ? parseItem(kind, body.data) : null;
  if (!item) throw new AdminContentError("server");
  return item;
}

function writeBody(kind: ContentKind, d: Draft, updatedAt?: string): string {
  const title = d.title.trim();
  const body: Record<string, unknown> =
    kind === "announcement"
      ? { title, body: d.text, status: d.status, pinned: d.pinned }
      : { title, description: d.text, status: d.status, topicId: d.topicId, sortOrder: d.sortOrder };
  if (updatedAt !== undefined) body.updatedAt = updatedAt;
  return JSON.stringify(body);
}

const writeHeaders = (csrf: string) => ({ Accept: "application/json", "Content-Type": "application/json", "X-CSRF-Token": csrf });

function requireValid(kind: ContentKind, d: Draft) {
  if (Object.keys(validateDraft(kind, d)).length > 0) throw new AdminContentError("validation");
}

function requireId(id: number) {
  if (!isInt(id, 1)) throw new AdminContentError("invalid_request");
}

/** Lists drafts and published items, newest first (server order). */
export async function listItems(
  kind: ContentKind,
  q: { page: number; limit: number },
  signal?: AbortSignal,
  deps: AdminContentDeps = {}
): Promise<{ items: AdminItem[]; meta: PageMeta }> {
  if (!isInt(q.page, 1) || !isInt(q.limit, 1) || q.limit > 100) throw new AdminContentError("invalid_request");
  const params = new URLSearchParams({ page: String(q.page), limit: String(q.limit) });
  const r = await send(`${PATHS[kind]}?${params}`, { method: "GET", headers: { Accept: "application/json" }, signal }, deps);
  if (!r.ok) throw await toError(r);
  const body = await readJson(r);
  if (!isRecord(body) || !Array.isArray(body.data) || !isMeta(body.meta)) throw new AdminContentError("server");
  const items = body.data.map((x) => parseItem(kind, x));
  if (items.some((x) => x === null)) throw new AdminContentError("server");
  return { items: items as AdminItem[], meta: body.meta };
}

export async function getItem(kind: ContentKind, id: number, signal?: AbortSignal, deps: AdminContentDeps = {}): Promise<AdminItem> {
  requireId(id);
  const r = await send(`${PATHS[kind]}/${id}`, { method: "GET", headers: { Accept: "application/json" }, signal }, deps);
  if (!r.ok) throw await toError(r);
  return readItem(kind, r);
}

export async function createItem(kind: ContentKind, d: Draft, csrf: string, deps: AdminContentDeps = {}): Promise<AdminItem> {
  requireValid(kind, d);
  const r = await send(PATHS[kind], { method: "POST", headers: writeHeaders(csrf), body: writeBody(kind, d) }, deps);
  if (!r.ok) throw await toError(r);
  return readItem(kind, r);
}

export async function updateItem(
  kind: ContentKind,
  id: number,
  d: Draft,
  updatedAt: string,
  csrf: string,
  deps: AdminContentDeps = {}
): Promise<AdminItem> {
  requireId(id);
  requireValid(kind, d);
  const r = await send(`${PATHS[kind]}/${id}`, { method: "PATCH", headers: writeHeaders(csrf), body: writeBody(kind, d, updatedAt) }, deps);
  if (!r.ok) throw await toError(r);
  return readItem(kind, r);
}

/** Publish or unpublish an item by resending it with a new status. */
export function setStatus(
  item: Draft & { kind: ContentKind; id: number; updatedAt: string },
  status: ContentStatus,
  csrf: string,
  deps: AdminContentDeps = {}
): Promise<AdminItem> {
  const { kind, id, updatedAt, ...draft } = item;
  return updateItem(kind, id, { ...draft, status }, updatedAt, csrf, deps);
}

export async function deleteItem(kind: ContentKind, id: number, csrf: string, deps: AdminContentDeps = {}): Promise<void> {
  requireId(id);
  const r = await send(`${PATHS[kind]}/${id}`, { method: "DELETE", headers: { Accept: "application/json", "X-CSRF-Token": csrf } }, deps);
  if (!r.ok) throw await toError(r);
}
