export const CONTENT_ERROR = "Konten tidak dapat dimuat. Silakan coba lagi.";
export type PageMeta = { page: number; limit: number; total: number; totalPages: number };
export type Announcement = { id: number; title: string; body: string; publishedAt: string };
export type Topic = { id: number; code: string; title: string; sortOrder: number };
export type MaterialTopic = Pick<Topic, "id" | "code" | "title">;
export type Material = { id: number; title: string; description: string; publishedAt: string; topic: MaterialTopic | null };
export type PageResult<T> = { items: T[]; meta: PageMeta };
type Fetcher = typeof fetch;
const record = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const integer = (v: unknown, min = 0): v is number => typeof v === "number" && Number.isSafeInteger(v) && v >= min;
const text = (v: unknown): v is string => typeof v === "string";
const date = (v: unknown): v is string => text(v) && v.trim() !== "" && !Number.isNaN(Date.parse(v));
const announcement = (v: unknown): v is Announcement => record(v) && integer(v.id, 1) && text(v.title) && text(v.body) && date(v.publishedAt);
const topic = (v: unknown): v is Topic => record(v) && integer(v.id, 1) && text(v.code) && text(v.title) && integer(v.sortOrder);
const materialTopic = (v: unknown): v is MaterialTopic => record(v) && integer(v.id, 1) && text(v.code) && text(v.title);
const material = (v: unknown): v is Material => record(v) && integer(v.id, 1) && text(v.title) && text(v.description) && date(v.publishedAt) && (v.topic === null || materialTopic(v.topic));
const meta = (v: unknown): v is PageMeta => record(v) && integer(v.page, 1) && integer(v.limit, 1) && integer(v.total) && integer(v.totalPages);
async function json(path: string, signal: AbortSignal | undefined, fetcher: Fetcher): Promise<unknown> {
  try { const r = await fetcher(path, { method: "GET", headers: { Accept: "application/json" }, cache: "no-store", signal }); if (!r.ok) throw new Error(); return await r.json(); }
  catch (error) { if (error instanceof DOMException && error.name === "AbortError") throw error; throw new Error(CONTENT_ERROR); }
}
async function list<T>(path: string, check: (v: unknown) => v is T, signal: AbortSignal | undefined, fetcher: Fetcher): Promise<PageResult<T>> {
  const payload = await json(path, signal, fetcher);
  if (!record(payload) || !Array.isArray(payload.data) || !payload.data.every(check) || !meta(payload.meta)) throw new Error(CONTENT_ERROR);
  return { items: payload.data, meta: payload.meta };
}
async function one<T>(path: string, check: (v: unknown) => v is T, signal: AbortSignal | undefined, fetcher: Fetcher): Promise<T> {
  const payload = await json(path, signal, fetcher); if (!record(payload) || !check(payload.data)) throw new Error(CONTENT_ERROR); return payload.data;
}
function params(q: { page: number; limit: number; topic?: string }) { if (!integer(q.page, 1) || !integer(q.limit, 1) || q.limit > 100) throw new Error(CONTENT_ERROR); const p = new URLSearchParams({ page: String(q.page), limit: String(q.limit) }); if (q.topic) p.set("topic", q.topic); return p; }
export const getAnnouncements = (q: {page:number;limit:number}, s?: AbortSignal, f: Fetcher = fetch) => list(`/api/v1/public/announcements?${params(q)}`, announcement, s, f);
export const getAnnouncement = (id: number, s?: AbortSignal, f: Fetcher = fetch) => integer(id, 1) ? one(`/api/v1/public/announcements/${id}`, announcement, s, f) : Promise.reject(new Error(CONTENT_ERROR));
export const getMaterials = (q: {page:number;limit:number;topic?:string}, s?: AbortSignal, f: Fetcher = fetch) => list(`/api/v1/public/materials?${params(q)}`, material, s, f);
export const getMaterial = (id: number, s?: AbortSignal, f: Fetcher = fetch) => integer(id, 1) ? one(`/api/v1/public/materials/${id}`, material, s, f) : Promise.reject(new Error(CONTENT_ERROR));
export async function getTopics(s?: AbortSignal, f: Fetcher = fetch): Promise<Topic[]> { const p = await json("/api/v1/public/topics", s, f); if (!record(p) || !Array.isArray(p.data) || !p.data.every(topic)) throw new Error(CONTENT_ERROR); return p.data; }
