import assert from "node:assert/strict";
import test from "node:test";

import {
  CONTENT_ERROR,
  getAnnouncement,
  getAnnouncements,
  getMaterial,
  getMaterials,
  getTopics
} from "./content.ts";

const announcement = { id: 7, title: "Ujian", body: "Besok", publishedAt: "2026-11-13T03:00:00Z" };
const material = { id: 9, title: "Jaringan", description: "Baca", publishedAt: "2026-11-12T03:00:00Z", topic: { id: 2, code: "jar", title: "Jaringan" } };
const response = (body: unknown, ok = true) => new Response(JSON.stringify(body), { status: ok ? 200 : 500 });

test("announcement list uses relative pagination URL and parses its envelope", async () => {
  let input = "";
  const fetcher: typeof fetch = async (value) => { input = String(value); return response({ data: [announcement], meta: { page: 2, limit: 5, total: 6, totalPages: 2 } }); };
  assert.deepEqual(await getAnnouncements({ page: 2, limit: 5 }, undefined, fetcher), { items: [announcement], meta: { page: 2, limit: 5, total: 6, totalPages: 2 } });
  assert.equal(input, "/api/v1/public/announcements?page=2&limit=5");
});

test("materials encodes topic filter and supports nullable topic", async () => {
  let input = "";
  const fetcher: typeof fetch = async (value) => { input = String(value); return response({ data: [{ ...material, topic: null }], meta: { page: 1, limit: 10, total: 1, totalPages: 1 } }); };
  const result = await getMaterials({ page: 1, limit: 10, topic: "web dasar" }, undefined, fetcher);
  assert.equal(input, "/api/v1/public/materials?page=1&limit=10&topic=web+dasar");
  assert.equal(result.items[0].topic, null);
});

test("detail and topics use relative URLs and propagate AbortSignal", async () => {
  const calls: Array<[string, AbortSignal | null | undefined]> = [];
  const fetcher: typeof fetch = async (value, init) => {
    calls.push([String(value), init?.signal]);
    if (String(value).endsWith("topics")) return response({ data: [{ id: 2, code: "jar", title: "Jaringan", sortOrder: 1 }] });
    return response({ data: String(value).includes("announcements") ? announcement : material });
  };
  const controller = new AbortController();
  await getAnnouncement(7, controller.signal, fetcher);
  await getMaterial(9, controller.signal, fetcher);
  await getTopics(controller.signal, fetcher);
  assert.deepEqual(calls.map(([url]) => url), ["/api/v1/public/announcements/7", "/api/v1/public/materials/9", "/api/v1/public/topics"]);
  assert.ok(calls.every(([, signal]) => signal === controller.signal));
});

test("rejects malformed and non-JSON responses with one safe fixed message", async () => {
  const malformed: typeof fetch = async () => response({ data: [{ ...announcement, id: "7" }], meta: { page: 1, limit: 10, total: 1, totalPages: 1 } });
  await assert.rejects(() => getAnnouncements({ page: 1, limit: 10 }, undefined, malformed), { message: CONTENT_ERROR });
  const broken: typeof fetch = async () => new Response("upstream secret", { status: 502 });
  await assert.rejects(() => getTopics(undefined, broken), { message: CONTENT_ERROR });
});

test("validates pagination before requesting", async () => {
  let called = false;
  const fetcher: typeof fetch = async () => { called = true; return response({}); };
  assert.throws(() => getMaterials({ page: 0, limit: 10 }, undefined, fetcher), { message: CONTENT_ERROR });
  assert.equal(called, false);
});
