// Typed client for material attachments (PDF/PPT/PPTX).
// Admin: list, upload (with progress), delete. Public: list and download URL.
// Server error text is never shown: every failure maps to a fixed message.
// The server decides the real file type from the content; the checks here
// only catch obvious mistakes before a large upload starts.

export type AttachmentKind = "pdf" | "ppt" | "pptx";

export const ATTACHMENT_LIMITS = {
  maxFile: 25 * 1024 * 1024,
  perMaterial: 5,
  extensions: [".pdf", ".ppt", ".pptx"] as const
} as const;

/** File input `accept` value. */
export const ACCEPT = ".pdf,.ppt,.pptx,application/pdf,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation";

export interface AdminAttachment {
  id: number;
  materialId: number;
  name: string;
  kind: AttachmentKind;
  size: number;
  createdAt: string;
}

export interface PublicAttachment {
  id: number;
  name: string;
  kind: AttachmentKind;
  size: number;
}

export type AttachmentErrorKind =
  | "unauthenticated"
  | "forbidden"
  | "not_found"
  | "limit"
  | "busy"
  | "too_large"
  | "empty"
  | "type"
  | "macro"
  | "quota"
  | "disk"
  | "name"
  | "invalid_request"
  | "network"
  | "server";

const MESSAGES: Record<AttachmentErrorKind, string> = {
  unauthenticated: "Sesi Anda telah berakhir. Silakan masuk kembali.",
  forbidden: "Permintaan ditolak oleh server. Muat ulang halaman, lalu coba lagi.",
  not_found: "Materi atau lampiran tidak ditemukan. Mungkin sudah dihapus.",
  limit: "Materi ini sudah punya 5 lampiran. Hapus satu dulu.",
  busy: "Unggahan lain sedang berjalan. Tunggu sebentar, lalu coba lagi.",
  too_large: "Berkas lebih dari 25 MB.",
  empty: "Berkas kosong.",
  type: "Hanya berkas PDF, PPT, atau PPTX yang diterima, dan isinya harus sesuai.",
  macro: "Presentasi bermakro tidak diterima. Simpan ulang sebagai .pptx biasa.",
  quota: "Kuota penyimpanan lampiran sudah penuh. Hapus lampiran lama dulu.",
  disk: "Ruang disk server tidak cukup. Hubungi pengelola server.",
  name: "Nama berkas tidak dapat dipakai. Ganti nama berkas, lalu coba lagi.",
  invalid_request: "Unggahan tidak valid atau terputus. Coba lagi.",
  network: "Tidak dapat terhubung ke server. Periksa koneksi Anda, lalu coba lagi.",
  server: "Terjadi gangguan pada server. Coba lagi beberapa saat lagi."
};

export function messageForAttachment(kind: AttachmentErrorKind): string {
  return MESSAGES[kind];
}

export class AttachmentError extends Error {
  kind: AttachmentErrorKind;
  constructor(kind: AttachmentErrorKind) {
    super(messageForAttachment(kind));
    this.name = "AttachmentError";
    this.kind = kind;
  }
}

const CODE_TO_KIND: Record<string, AttachmentErrorKind> = {
  UNAUTHENTICATED: "unauthenticated",
  FORBIDDEN: "forbidden",
  NOT_FOUND: "not_found",
  ATTACHMENT_LIMIT: "limit",
  SERVICE_BUSY: "busy",
  PAYLOAD_TOO_LARGE: "too_large",
  UNSUPPORTED_FILE_TYPE: "type",
  MACRO_NOT_ALLOWED: "macro",
  STORAGE_QUOTA_EXCEEDED: "quota",
  INSUFFICIENT_STORAGE: "disk",
  INVALID_FILE_NAME: "name",
  INVALID_REQUEST: "invalid_request",
  UNSUPPORTED_MEDIA_TYPE: "invalid_request"
};

const STATUS_TO_KIND: Record<number, AttachmentErrorKind> = {
  401: "unauthenticated",
  403: "forbidden",
  404: "not_found",
  409: "limit",
  413: "too_large",
  415: "type",
  503: "busy",
  507: "quota"
};

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const isInt = (v: unknown, min: number): v is number => typeof v === "number" && Number.isSafeInteger(v) && v >= min;
const isDate = (v: unknown): v is string => typeof v === "string" && v.trim() !== "" && !Number.isNaN(Date.parse(v));
const isKind = (v: unknown): v is AttachmentKind => v === "pdf" || v === "ppt" || v === "pptx";
const isName = (v: unknown): v is string => typeof v === "string" && v.trim() !== "" && [...v].length <= 200;
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";

function parseAdmin(v: unknown): AdminAttachment | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isInt(v.materialId, 1) || !isName(v.name) || !isKind(v.kind)) return null;
  if (!isInt(v.size, 1) || v.size > ATTACHMENT_LIMITS.maxFile || !isDate(v.createdAt)) return null;
  return { id: v.id, materialId: v.materialId, name: v.name, kind: v.kind, size: v.size, createdAt: v.createdAt };
}

function parsePublic(v: unknown): PublicAttachment | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isName(v.name) || !isKind(v.kind) || !isInt(v.size, 1)) return null;
  return { id: v.id, name: v.name, kind: v.kind, size: v.size };
}

function errorFrom(status: number, body: unknown): AttachmentError {
  const code = isRecord(body) && isRecord(body.error) && typeof body.error.code === "string" ? body.error.code : "";
  return new AttachmentError(CODE_TO_KIND[code] ?? STATUS_TO_KIND[status] ?? "server");
}

function parseText(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}

/** Checks name and size before uploading. Returns null when the file may be sent. */
export function checkFile(file: { name: string; size: number }): AttachmentErrorKind | null {
  const lower = file.name.toLowerCase();
  if (!ATTACHMENT_LIMITS.extensions.some((ext) => lower.endsWith(ext))) return "type";
  if (file.size <= 0) return "empty";
  if (file.size > ATTACHMENT_LIMITS.maxFile) return "too_large";
  return null;
}

/** "1,5 MB", "820 KB", "12 B" (binary units, matching the 25 MB limit). */
export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024)).toLocaleString("id-ID")} KB`;
  return `${(bytes / (1024 * 1024)).toLocaleString("id-ID", { maximumFractionDigits: 1 })} MB`;
}

export const KIND_LABEL: Record<AttachmentKind, string> = { pdf: "PDF", ppt: "PPT", pptx: "PPTX" };

export function downloadURL(id: number): string {
  if (!isInt(id, 1)) throw new AttachmentError("invalid_request");
  return `/api/v1/public/attachments/${id}`;
}

// ---- transport ---------------------------------------------------------------

export interface UploadRequest {
  url: string;
  body: FormData;
  headers: Record<string, string>;
  onProgress?: (sent: number, total: number) => void;
  signal?: AbortSignal;
}

/** Sends a multipart upload and resolves with the raw response. Rejects with
 *  AbortError on cancel and TypeError on network failure. */
export type UploadTransport = (req: UploadRequest) => Promise<{ status: number; text: string }>;

/** XMLHttpRequest transport: fetch cannot report upload progress. */
export const xhrTransport: UploadTransport = (req) =>
  new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", req.url);
    xhr.withCredentials = true;
    for (const [k, v] of Object.entries(req.headers)) xhr.setRequestHeader(k, v);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) req.onProgress?.(e.loaded, e.total);
    };
    xhr.onload = () => resolve({ status: xhr.status, text: xhr.responseText });
    xhr.onerror = () => reject(new TypeError("network"));
    xhr.ontimeout = () => reject(new TypeError("network"));
    xhr.onabort = () => reject(new DOMException("Unggahan dibatalkan.", "AbortError"));
    if (req.signal) {
      if (req.signal.aborted) {
        reject(new DOMException("Unggahan dibatalkan.", "AbortError"));
        return;
      }
      req.signal.addEventListener("abort", () => xhr.abort(), { once: true });
    }
    xhr.send(req.body);
  });

export interface AttachmentDeps {
  fetch?: typeof fetch;
  transport?: UploadTransport;
}

async function get(url: string, signal: AbortSignal | undefined, deps: AttachmentDeps): Promise<{ status: number; body: unknown }> {
  const doFetch = deps.fetch ?? fetch;
  let r: Response;
  try {
    r = await doFetch(url, { method: "GET", headers: { Accept: "application/json" }, credentials: "same-origin", cache: "no-store", signal });
  } catch (e) {
    if (isAbort(e)) throw e;
    throw new AttachmentError("network");
  }
  let body: unknown;
  try {
    body = await r.json();
  } catch {
    body = undefined;
  }
  return { status: r.status, body };
}

function requireId(id: number) {
  if (!isInt(id, 1)) throw new AttachmentError("invalid_request");
}

// ---- admin -------------------------------------------------------------------

export async function listAdminAttachments(materialId: number, signal?: AbortSignal, deps: AttachmentDeps = {}): Promise<AdminAttachment[]> {
  requireId(materialId);
  const { status, body } = await get(`/api/v1/admin/materials/${materialId}/attachments`, signal, deps);
  if (status !== 200) throw errorFrom(status, body);
  if (!isRecord(body) || !Array.isArray(body.data)) throw new AttachmentError("server");
  const items = body.data.map(parseAdmin);
  if (items.some((x) => x === null)) throw new AttachmentError("server");
  return items as AdminAttachment[];
}

export async function uploadAttachment(
  materialId: number,
  file: File,
  csrf: string,
  opts: { onProgress?: (sent: number, total: number) => void; signal?: AbortSignal } = {},
  deps: AttachmentDeps = {}
): Promise<AdminAttachment> {
  requireId(materialId);
  const problem = checkFile(file);
  if (problem) throw new AttachmentError(problem);
  const body = new FormData();
  body.append("file", file, file.name);
  const transport = deps.transport ?? xhrTransport;
  let res: { status: number; text: string };
  try {
    res = await transport({
      url: `/api/v1/admin/materials/${materialId}/attachments`,
      body,
      headers: { Accept: "application/json", "X-CSRF-Token": csrf },
      onProgress: opts.onProgress,
      signal: opts.signal
    });
  } catch (e) {
    if (isAbort(e)) throw e;
    throw new AttachmentError("network");
  }
  const parsed = parseText(res.text);
  if (res.status !== 201) throw errorFrom(res.status, parsed);
  const item = isRecord(parsed) ? parseAdmin(parsed.data) : null;
  if (!item) throw new AttachmentError("server");
  return item;
}

export async function deleteAttachment(id: number, csrf: string, deps: AttachmentDeps = {}): Promise<void> {
  requireId(id);
  const doFetch = deps.fetch ?? fetch;
  let r: Response;
  try {
    r = await doFetch(`/api/v1/admin/attachments/${id}`, {
      method: "DELETE",
      headers: { Accept: "application/json", "X-CSRF-Token": csrf },
      credentials: "same-origin",
      cache: "no-store"
    });
  } catch {
    throw new AttachmentError("network");
  }
  if (r.status === 204) return;
  let body: unknown;
  try {
    body = await r.json();
  } catch {
    body = undefined;
  }
  throw errorFrom(r.status, body);
}

// ---- public ------------------------------------------------------------------

export async function getPublicAttachments(materialId: number, signal?: AbortSignal, deps: AttachmentDeps = {}): Promise<PublicAttachment[]> {
  requireId(materialId);
  const { status, body } = await get(`/api/v1/public/materials/${materialId}/attachments`, signal, deps);
  if (status !== 200) throw errorFrom(status, body);
  if (!isRecord(body) || !Array.isArray(body.data)) throw new AttachmentError("server");
  const items = body.data.map(parsePublic);
  if (items.some((x) => x === null)) throw new AttachmentError("server");
  return items as PublicAttachment[];
}
