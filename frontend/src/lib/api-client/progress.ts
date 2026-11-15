// Typed client for the Progres Tugas Kelompok API.
// Public: the progress table. Admin: academic years, classes, and groups.
// Every write sends the CSRF token; server error text is never shown.

export type Submission = "BELUM" | "SEBAGIAN" | "LENGKAP";
export type Review = "BELUM_DICEK" | "DIPERIKSA" | "PERLU_REVISI" | "REVISI_DITERIMA" | "DISETUJUI";
export type Notebook = "BELUM_DITERIMA" | "SUDAH_DITERIMA";
export type StatusFilter = "" | Submission | "TUNTAS";

export const PROGRESS_LIMITS = { classCode: 20, className: 80, groupName: 60, email: 254, query: 60 } as const;

export const SUBMISSION_LABEL: Record<Submission, string> = {
  BELUM: "Belum Mengumpulkan",
  SEBAGIAN: "Terkumpul Sebagian",
  LENGKAP: "Berkas Digital Lengkap"
};
export const REVIEW_LABEL: Record<Review, string> = {
  BELUM_DICEK: "Belum Diperiksa",
  DIPERIKSA: "Sedang Diperiksa",
  PERLU_REVISI: "Perlu Revisi",
  REVISI_DITERIMA: "Revisi Diterima, Menunggu Pemeriksaan",
  DISETUJUI: "Materi Disetujui"
};
export const NOTEBOOK_LABEL: Record<Notebook, string> = {
  BELUM_DITERIMA: "Belum Diterima",
  SUDAH_DITERIMA: "Sudah Diterima"
};

export interface PublicGroup {
  name: string;
  classCode: string;
  topic: { code: string; title: string };
  submission: Submission;
  review: Review;
  notebook: Notebook;
  complete: boolean;
  updatedAt: string;
}
export interface PublicProgress {
  year: string;
  classes: { code: string; name: string }[];
  groups: PublicGroup[];
}

export interface AcademicYear {
  id: number;
  label: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}
export interface SchoolClass {
  id: number;
  yearId: number;
  code: string;
  name: string;
  active: boolean;
  groupCount: number;
  createdAt: string;
  updatedAt: string;
}
export interface StudyGroup {
  id: number;
  classId: number;
  topicId: number;
  name: string;
  email: string | null;
  sortOrder: number;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}
export interface ClassDraft {
  yearId: number;
  code: string;
  name: string;
  active: boolean;
}
export interface GroupDraft {
  classId: number;
  topicId: number;
  name: string;
  email: string;
  sortOrder: number;
  active: boolean;
}

export type ProgressErrorKind =
  | "unauthenticated"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "duplicate"
  | "in_use"
  | "not_last"
  | "reference"
  | "validation"
  | "invalid_request"
  | "network"
  | "server";

const MESSAGES: Record<ProgressErrorKind, string> = {
  unauthenticated: "Sesi Anda telah berakhir. Silakan masuk kembali.",
  forbidden: "Permintaan ditolak oleh server. Muat ulang halaman, lalu coba lagi.",
  not_found: "Data tidak ditemukan. Mungkin sudah dihapus.",
  conflict: "Data ini sudah diubah di tempat lain. Daftar dimuat ulang; periksa lalu ulangi.",
  duplicate: "Nama atau kode itu sudah dipakai (huruf besar/kecil dianggap sama).",
  in_use: "Masih punya data terkait, jadi tidak bisa dihapus. Nonaktifkan saja.",
  not_last: "Hanya putaran terakhir yang bisa dihapus. Koreksi putaran lama lewat Ubah.",
  reference: "Tahun ajaran, kelas, atau topik yang dipilih tidak ada. Muat ulang halaman.",
  validation: "Isian belum sesuai aturan. Periksa lalu coba lagi.",
  invalid_request: "Permintaan tidak valid. Muat ulang halaman, lalu coba lagi.",
  network: "Tidak dapat terhubung ke server. Periksa koneksi Anda, lalu coba lagi.",
  server: "Terjadi gangguan pada server. Coba lagi beberapa saat lagi."
};

export function messageForProgress(kind: ProgressErrorKind): string {
  return MESSAGES[kind];
}

export class ProgressError extends Error {
  kind: ProgressErrorKind;
  constructor(kind: ProgressErrorKind) {
    super(messageForProgress(kind));
    this.name = "ProgressError";
    this.kind = kind;
  }
}

const CODE_TO_KIND: Record<string, ProgressErrorKind> = {
  UNAUTHENTICATED: "unauthenticated",
  FORBIDDEN: "forbidden",
  NOT_FOUND: "not_found",
  CONFLICT: "conflict",
  DUPLICATE: "duplicate",
  IN_USE: "in_use",
  NOT_LAST_ROUND: "not_last",
  INVALID_REFERENCE: "reference",
  VALIDATION_ERROR: "validation",
  INVALID_REQUEST: "invalid_request",
  UNSUPPORTED_MEDIA_TYPE: "invalid_request",
  PAYLOAD_TOO_LARGE: "invalid_request"
};
const STATUS_TO_KIND: Record<number, ProgressErrorKind> = {
  400: "invalid_request",
  401: "unauthenticated",
  403: "forbidden",
  404: "not_found",
  409: "conflict",
  422: "validation"
};

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const isInt = (v: unknown, min: number): v is number => typeof v === "number" && Number.isSafeInteger(v) && v >= min;
const isDate = (v: unknown): v is string => typeof v === "string" && v.trim() !== "" && !Number.isNaN(Date.parse(v));
const isStr = (v: unknown): v is string => typeof v === "string";
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";
const chars = (s: string) => [...s].length;
const oneOf = <T extends string>(v: unknown, xs: readonly T[]): v is T => typeof v === "string" && (xs as readonly string[]).includes(v);

const SUBMISSIONS = ["BELUM", "SEBAGIAN", "LENGKAP"] as const;
const REVIEWS = ["BELUM_DICEK", "DIPERIKSA", "PERLU_REVISI", "REVISI_DITERIMA", "DISETUJUI"] as const;
const NOTEBOOKS = ["BELUM_DITERIMA", "SUDAH_DITERIMA"] as const;

function parsePublicGroup(v: unknown): PublicGroup | null {
  if (!isRecord(v) || !isStr(v.name) || !isStr(v.classCode) || !isRecord(v.topic)) return null;
  if (!isStr(v.topic.code) || !isStr(v.topic.title)) return null;
  if (!oneOf(v.submission, SUBMISSIONS) || !oneOf(v.review, REVIEWS) || !oneOf(v.notebook, NOTEBOOKS)) return null;
  if (typeof v.complete !== "boolean" || !isDate(v.updatedAt)) return null;
  return {
    name: v.name,
    classCode: v.classCode,
    topic: { code: v.topic.code, title: v.topic.title },
    submission: v.submission,
    review: v.review,
    notebook: v.notebook,
    complete: v.complete,
    updatedAt: v.updatedAt
  };
}

function parseYear(v: unknown): AcademicYear | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isStr(v.label) || typeof v.active !== "boolean" || !isDate(v.createdAt) || !isDate(v.updatedAt)) return null;
  return { id: v.id, label: v.label, active: v.active, createdAt: v.createdAt, updatedAt: v.updatedAt };
}

function parseClass(v: unknown): SchoolClass | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isInt(v.yearId, 1) || !isStr(v.code) || !isStr(v.name)) return null;
  if (typeof v.active !== "boolean" || !isInt(v.groupCount, 0) || !isDate(v.createdAt) || !isDate(v.updatedAt)) return null;
  return { id: v.id, yearId: v.yearId, code: v.code, name: v.name, active: v.active, groupCount: v.groupCount, createdAt: v.createdAt, updatedAt: v.updatedAt };
}

function parseGroup(v: unknown): StudyGroup | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isInt(v.classId, 1) || !isInt(v.topicId, 1) || !isStr(v.name)) return null;
  if (!(v.email === null || isStr(v.email)) || !isInt(v.sortOrder, 0) || typeof v.active !== "boolean") return null;
  if (!isDate(v.createdAt) || !isDate(v.updatedAt)) return null;
  return {
    id: v.id,
    classId: v.classId,
    topicId: v.topicId,
    name: v.name,
    email: v.email,
    sortOrder: v.sortOrder,
    active: v.active,
    createdAt: v.createdAt,
    updatedAt: v.updatedAt
  };
}

export interface ProgressDeps {
  fetch?: typeof fetch;
}

async function call(url: string, init: RequestInit, deps: ProgressDeps): Promise<{ status: number; body: unknown }> {
  const doFetch = deps.fetch ?? fetch;
  let r: Response;
  try {
    r = await doFetch(url, { ...init, credentials: "same-origin", cache: "no-store" });
  } catch (e) {
    if (isAbort(e)) throw e;
    throw new ProgressError("network");
  }
  if (r.status === 204) return { status: 204, body: undefined };
  let body: unknown;
  try {
    body = await r.json();
  } catch {
    body = undefined;
  }
  return { status: r.status, body };
}

function errorFrom(status: number, body: unknown): ProgressError {
  const code = isRecord(body) && isRecord(body.error) && typeof body.error.code === "string" ? body.error.code : "";
  return new ProgressError(CODE_TO_KIND[code] ?? STATUS_TO_KIND[status] ?? "server");
}

function data<T>(res: { status: number; body: unknown }, ok: number, parse: (v: unknown) => T | null): T {
  if (res.status !== ok) throw errorFrom(res.status, res.body);
  const v = isRecord(res.body) ? parse(res.body.data) : null;
  if (v === null) throw new ProgressError("server");
  return v;
}

function list<T>(parse: (v: unknown) => T | null) {
  return (v: unknown): T[] | null => {
    if (!Array.isArray(v)) return null;
    const out = v.map(parse);
    return out.some((x) => x === null) ? null : (out as T[]);
  };
}

const get = (url: string, signal: AbortSignal | undefined, deps: ProgressDeps) =>
  call(url, { method: "GET", headers: { Accept: "application/json" }, signal }, deps);

const write = (method: string, url: string, csrf: string, body: unknown, deps: ProgressDeps) =>
  call(
    url,
    {
      method,
      headers: body === undefined ? { Accept: "application/json", "X-CSRF-Token": csrf } : { Accept: "application/json", "Content-Type": "application/json", "X-CSRF-Token": csrf },
      body: body === undefined ? undefined : JSON.stringify(body)
    },
    deps
  );

function requireId(id: number) {
  if (!isInt(id, 1)) throw new ProgressError("invalid_request");
}

// ---- validation (mirrors the Go rules) ---------------------------------------

export function validateYearLabel(label: string): string | null {
  const m = /^(\d{4})\/(\d{4})$/.exec(label.trim());
  if (!m || Number(m[2]) !== Number(m[1]) + 1) return "Tulis tahun ajaran seperti 2027/2028.";
  return null;
}

export type ClassErrors = Partial<Record<"code" | "name", string>>;
export function validateClass(d: ClassDraft): ClassErrors {
  const e: ClassErrors = {};
  const code = d.code.trim();
  if (code === "" || chars(code) > PROGRESS_LIMITS.classCode || !/^[0-9A-Za-z][0-9A-Za-z .-]*$/.test(code)) {
    e.code = "Kode kelas 1–20 karakter: huruf, angka, spasi, titik, atau strip (mis. 9A).";
  }
  if (chars(d.name.trim()) > PROGRESS_LIMITS.className) e.name = "Nama kelas paling banyak 80 karakter.";
  return e;
}

export type GroupErrors = Partial<Record<"name" | "email" | "topicId" | "sortOrder", string>>;
const EMAIL = /^[a-z0-9._%+-]+@[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$/;
export function validateGroup(d: GroupDraft): GroupErrors {
  const e: GroupErrors = {};
  const name = d.name.trim();
  if (name === "" || chars(name) > PROGRESS_LIMITS.groupName || /[\u0000-\u001f\u007f]/.test(name)) e.name = "Nama kelompok wajib, paling banyak 60 karakter.";
  const email = d.email.trim().toLowerCase();
  if (email !== "" && (email.length > PROGRESS_LIMITS.email || !EMAIL.test(email))) e.email = "Alamat email tidak valid.";
  if (!isInt(d.topicId, 1)) e.topicId = "Pilih topik.";
  if (!Number.isInteger(d.sortOrder) || d.sortOrder < 0) e.sortOrder = "Urutan harus bilangan bulat 0 atau lebih.";
  return e;
}

/** Suggested group email from the juknis format, e.g. kelompok1.9a.2627.informatika@gmail.com. */
export function suggestEmail(groupName: string, classCode: string, yearLabel: string): string {
  const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, "");
  const m = /^(\d{2})(\d{2})\/(\d{2})(\d{2})$/.exec(yearLabel);
  const year = m ? `${m[2]}${m[4]}` : "";
  const g = slug(groupName);
  const c = slug(classCode);
  if (!g || !c || !year) return "";
  return `${g}.${c}.${year}.informatika@gmail.com`;
}

// ---- public ------------------------------------------------------------------

export async function getPublicProgress(
  f: { class?: string; q?: string; status?: StatusFilter } = {},
  signal?: AbortSignal,
  deps: ProgressDeps = {}
): Promise<PublicProgress> {
  const p = new URLSearchParams();
  if (f.class) p.set("class", f.class);
  if (f.q && f.q.trim()) p.set("q", f.q.trim().slice(0, PROGRESS_LIMITS.query));
  if (f.status) p.set("status", f.status);
  const qs = p.toString();
  const res = await get(`/api/v1/public/progress${qs ? `?${qs}` : ""}`, signal, deps);
  return data(res, 200, (v) => {
    if (!isRecord(v) || !isStr(v.year) || !Array.isArray(v.classes) || !Array.isArray(v.groups)) return null;
    const classes = v.classes.map((c) => (isRecord(c) && isStr(c.code) && isStr(c.name) ? { code: c.code, name: c.name } : null));
    const groups = v.groups.map(parsePublicGroup);
    if (classes.some((x) => x === null) || groups.some((x) => x === null)) return null;
    return { year: v.year, classes: classes as PublicProgress["classes"], groups: groups as PublicGroup[] };
  });
}

// ---- admin -------------------------------------------------------------------

export async function listYears(signal?: AbortSignal, deps: ProgressDeps = {}): Promise<AcademicYear[]> {
  return data(await get("/api/v1/admin/academic-years", signal, deps), 200, list(parseYear));
}

export async function createYear(label: string, csrf: string, deps: ProgressDeps = {}): Promise<AcademicYear> {
  if (validateYearLabel(label)) throw new ProgressError("validation");
  return data(await write("POST", "/api/v1/admin/academic-years", csrf, { label: label.trim() }, deps), 201, parseYear);
}

export async function activateYear(id: number, csrf: string, deps: ProgressDeps = {}): Promise<void> {
  requireId(id);
  const res = await write("POST", `/api/v1/admin/academic-years/${id}/activate`, csrf, undefined, deps);
  if (res.status !== 204) throw errorFrom(res.status, res.body);
}

export async function listClasses(yearId: number | null, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<SchoolClass[]> {
  if (yearId !== null) requireId(yearId);
  const url = yearId === null ? "/api/v1/admin/classes" : `/api/v1/admin/classes?year=${yearId}`;
  return data(await get(url, signal, deps), 200, list(parseClass));
}

export async function getClass(id: number, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<SchoolClass> {
  requireId(id);
  return data(await get(`/api/v1/admin/classes/${id}`, signal, deps), 200, parseClass);
}

const classBody = (d: ClassDraft) => ({ yearId: d.yearId, code: d.code.trim(), name: d.name.trim() });

export async function createClass(d: ClassDraft, csrf: string, deps: ProgressDeps = {}): Promise<SchoolClass> {
  if (Object.keys(validateClass(d)).length) throw new ProgressError("validation");
  return data(await write("POST", "/api/v1/admin/classes", csrf, classBody(d), deps), 201, parseClass);
}

export async function updateClass(id: number, d: ClassDraft, updatedAt: string, csrf: string, deps: ProgressDeps = {}): Promise<SchoolClass> {
  requireId(id);
  if (Object.keys(validateClass(d)).length) throw new ProgressError("validation");
  return data(await write("PATCH", `/api/v1/admin/classes/${id}`, csrf, { ...classBody(d), active: d.active, updatedAt }, deps), 200, parseClass);
}

export async function deleteClass(id: number, csrf: string, deps: ProgressDeps = {}): Promise<void> {
  requireId(id);
  const res = await write("DELETE", `/api/v1/admin/classes/${id}`, csrf, undefined, deps);
  if (res.status !== 204) throw errorFrom(res.status, res.body);
}

export async function listGroups(classId: number, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<StudyGroup[]> {
  requireId(classId);
  return data(await get(`/api/v1/admin/groups?class=${classId}`, signal, deps), 200, list(parseGroup));
}

const groupBody = (d: GroupDraft) => ({
  classId: d.classId,
  topicId: d.topicId,
  name: d.name.trim(),
  email: d.email.trim().toLowerCase(),
  sortOrder: d.sortOrder
});

export async function createGroup(d: GroupDraft, csrf: string, deps: ProgressDeps = {}): Promise<StudyGroup> {
  if (Object.keys(validateGroup(d)).length) throw new ProgressError("validation");
  return data(await write("POST", "/api/v1/admin/groups", csrf, groupBody(d), deps), 201, parseGroup);
}

export async function updateGroup(id: number, d: GroupDraft, updatedAt: string, csrf: string, deps: ProgressDeps = {}): Promise<StudyGroup> {
  requireId(id);
  if (Object.keys(validateGroup(d)).length) throw new ProgressError("validation");
  return data(await write("PATCH", `/api/v1/admin/groups/${id}`, csrf, { ...groupBody(d), active: d.active, updatedAt }, deps), 200, parseGroup);
}

export async function deleteGroup(id: number, csrf: string, deps: ProgressDeps = {}): Promise<void> {
  requireId(id);
  const res = await write("DELETE", `/api/v1/admin/groups/${id}`, csrf, undefined, deps);
  if (res.status !== 204) throw errorFrom(res.status, res.body);
}

// ---- submissions and notebook (3B) -------------------------------------------

export interface SubmissionRound {
  id: number;
  groupId: number;
  roundNumber: number;
  receivedAt: string;
  senderEmail: string | null;
  subject: string;
  ppt: boolean;
  pdf: boolean;
  pptFilename: string;
  pdfFilename: string;
  note: string;
  createdAt: string;
  updatedAt: string;
}

export interface RoundDraft {
  /** Wall-clock time in WIB as typed in a datetime-local input: YYYY-MM-DDTHH:mm. */
  receivedAt: string;
  senderEmail: string;
  subject: string;
  ppt: boolean;
  pdf: boolean;
  pptFilename: string;
  pdfFilename: string;
  note: string;
}

export interface GroupNotebook {
  received: boolean;
  receivedAt: string | null;
  note: string;
  updatedAt: string | null;
}

export const ROUND_LIMITS = { subject: 300, fileName: 200, note: 2000, notebookNote: 1000 } as const;

const WIB_OFFSET_MS = 7 * 60 * 60 * 1000;
const LOCAL = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/;

/** ISO instant -> "YYYY-MM-DDTHH:mm" in WIB, for a datetime-local input. */
export function toWIBInput(iso: string | Date): string {
  const d = new Date(new Date(iso).getTime() + WIB_OFFSET_MS);
  return d.toISOString().slice(0, 16);
}

/** "YYYY-MM-DDTHH:mm" typed in WIB -> ISO instant with +07:00, or null if malformed. */
export function fromWIBInput(local: string): string | null {
  const m = LOCAL.exec(local.trim());
  if (!m) return null;
  const iso = `${m[1]}-${m[2]}-${m[3]}T${m[4]}:${m[5]}:00+07:00`;
  const t = Date.parse(iso);
  if (Number.isNaN(t) || toWIBInput(new Date(t)) !== local.trim()) return null;
  return iso;
}

/** "Awal" for round 0, "Revisi n" afterwards. */
export function roundLabel(n: number): string {
  return n === 0 ? "Pengumpulan awal" : `Revisi ${n}`;
}

/** Suggested subject from the juknis, e.g. "REVISI 1 - Proyek Informatika IX - 9A - Kelompok 1 - Struktur Data". */
export function suggestSubject(round: number, classCode: string, groupName: string, topicTitle: string): string {
  const topic = topicTitle.replace(/\s*\(.*\)\s*$/, "").trim();
  const base = `Proyek Informatika IX - ${classCode} - ${groupName} - ${topic}`;
  return round === 0 ? base : `REVISI ${round} - ${base}`;
}

export type RoundErrors = Partial<Record<"receivedAt" | "files" | "pptFilename" | "pdfFilename" | "senderEmail" | "subject" | "note", string>>;

const CONTROL = /[\u0000-\u001f\u007f]/;
const badFileName = (s: string) => chars(s) > ROUND_LIMITS.fileName || CONTROL.test(s) || /[/\\]/.test(s);

export function validateRound(d: RoundDraft, now: Date = new Date()): RoundErrors {
  const e: RoundErrors = {};
  const iso = fromWIBInput(d.receivedAt);
  if (!iso) e.receivedAt = "Isi tanggal dan jam email diterima (WIB).";
  else if (Date.parse(iso) > now.getTime() + 5 * 60 * 1000) e.receivedAt = "Waktu terima tidak boleh di masa depan.";
  if (!d.ppt && !d.pdf) e.files = "Centang PPT, PDF, atau keduanya.";
  if (!d.ppt && d.pptFilename.trim() !== "") e.pptFilename = "Kosongkan nama berkas PPT, atau centang PPT.";
  else if (badFileName(d.pptFilename.trim())) e.pptFilename = "Nama berkas paling banyak 200 karakter, tanpa / atau \\.";
  if (!d.pdf && d.pdfFilename.trim() !== "") e.pdfFilename = "Kosongkan nama berkas PDF, atau centang PDF.";
  else if (badFileName(d.pdfFilename.trim())) e.pdfFilename = "Nama berkas paling banyak 200 karakter, tanpa / atau \\.";
  const sender = d.senderEmail.trim().toLowerCase();
  if (sender !== "" && (sender.length > PROGRESS_LIMITS.email || !EMAIL.test(sender))) e.senderEmail = "Alamat pengirim tidak valid.";
  const subject = d.subject.trim();
  if (chars(subject) > ROUND_LIMITS.subject || CONTROL.test(subject)) e.subject = "Subjek paling banyak 300 karakter, satu baris.";
  if (chars(d.note.trim()) > ROUND_LIMITS.note) e.note = "Catatan paling banyak 2.000 karakter.";
  return e;
}

function parseRound(v: unknown): SubmissionRound | null {
  if (!isRecord(v) || !isInt(v.id, 1) || !isInt(v.groupId, 1) || !isInt(v.roundNumber, 0) || !isDate(v.receivedAt)) return null;
  if (!(v.senderEmail === null || isStr(v.senderEmail)) || !isStr(v.subject) || typeof v.ppt !== "boolean" || typeof v.pdf !== "boolean") return null;
  if (!isStr(v.pptFilename) || !isStr(v.pdfFilename) || !isStr(v.note) || !isDate(v.createdAt) || !isDate(v.updatedAt)) return null;
  return {
    id: v.id,
    groupId: v.groupId,
    roundNumber: v.roundNumber,
    receivedAt: v.receivedAt,
    senderEmail: v.senderEmail,
    subject: v.subject,
    ppt: v.ppt,
    pdf: v.pdf,
    pptFilename: v.pptFilename,
    pdfFilename: v.pdfFilename,
    note: v.note,
    createdAt: v.createdAt,
    updatedAt: v.updatedAt
  };
}

function parseNotebook(v: unknown): GroupNotebook | null {
  if (!isRecord(v) || typeof v.received !== "boolean" || !isStr(v.note)) return null;
  if (!(v.receivedAt === null || isDate(v.receivedAt)) || !(v.updatedAt === null || isDate(v.updatedAt))) return null;
  if (v.received !== (v.receivedAt !== null)) return null;
  return { received: v.received, receivedAt: v.receivedAt, note: v.note, updatedAt: v.updatedAt };
}

const roundBody = (d: RoundDraft) => ({
  receivedAt: fromWIBInput(d.receivedAt),
  senderEmail: d.senderEmail.trim().toLowerCase(),
  subject: d.subject.trim(),
  ppt: d.ppt,
  pdf: d.pdf,
  pptFilename: d.ppt ? d.pptFilename.trim() : "",
  pdfFilename: d.pdf ? d.pdfFilename.trim() : "",
  note: d.note.trim()
});

export const draftFromRound = (r: SubmissionRound): RoundDraft => ({
  receivedAt: toWIBInput(r.receivedAt),
  senderEmail: r.senderEmail ?? "",
  subject: r.subject,
  ppt: r.ppt,
  pdf: r.pdf,
  pptFilename: r.pptFilename,
  pdfFilename: r.pdfFilename,
  note: r.note
});

export async function getGroup(id: number, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<StudyGroup> {
  requireId(id);
  return data(await get(`/api/v1/admin/groups/${id}`, signal, deps), 200, parseGroup);
}

export async function listRounds(groupId: number, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<{ submission: Submission; rounds: SubmissionRound[] }> {
  requireId(groupId);
  return data(await get(`/api/v1/admin/groups/${groupId}/submissions`, signal, deps), 200, (v) => {
    if (!isRecord(v) || !oneOf(v.submission, SUBMISSIONS)) return null;
    const rounds = list(parseRound)(v.rounds);
    return rounds ? { submission: v.submission, rounds } : null;
  });
}

export async function recordRound(groupId: number, roundNumber: number, d: RoundDraft, csrf: string, deps: ProgressDeps = {}): Promise<SubmissionRound> {
  requireId(groupId);
  if (!isInt(roundNumber, 0) || Object.keys(validateRound(d)).length) throw new ProgressError("validation");
  return data(await write("POST", `/api/v1/admin/groups/${groupId}/submissions`, csrf, { roundNumber, ...roundBody(d) }, deps), 201, parseRound);
}

export async function updateRound(id: number, d: RoundDraft, updatedAt: string, csrf: string, deps: ProgressDeps = {}): Promise<SubmissionRound> {
  requireId(id);
  if (Object.keys(validateRound(d)).length) throw new ProgressError("validation");
  return data(await write("PATCH", `/api/v1/admin/submissions/${id}`, csrf, { ...roundBody(d), updatedAt }, deps), 200, parseRound);
}

export async function deleteRound(id: number, csrf: string, deps: ProgressDeps = {}): Promise<void> {
  requireId(id);
  const res = await write("DELETE", `/api/v1/admin/submissions/${id}`, csrf, undefined, deps);
  if (res.status !== 204) throw errorFrom(res.status, res.body);
}

export async function getNotebook(groupId: number, signal?: AbortSignal, deps: ProgressDeps = {}): Promise<GroupNotebook> {
  requireId(groupId);
  return data(await get(`/api/v1/admin/groups/${groupId}/notebook`, signal, deps), 200, parseNotebook);
}

export async function putNotebook(
  groupId: number,
  d: { received: boolean; receivedAt: string; note: string },
  csrf: string,
  deps: ProgressDeps = {}
): Promise<GroupNotebook> {
  requireId(groupId);
  const at = d.received && d.receivedAt.trim() !== "" ? fromWIBInput(d.receivedAt) : null;
  if ((d.received && d.receivedAt.trim() !== "" && !at) || chars(d.note.trim()) > ROUND_LIMITS.notebookNote) throw new ProgressError("validation");
  return data(await write("PUT", `/api/v1/admin/groups/${groupId}/notebook`, csrf, { received: d.received, receivedAt: at, note: d.note.trim() }, deps), 200, parseNotebook);
}
