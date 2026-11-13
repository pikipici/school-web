import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  AdminContentError,
  LIMITS,
  createItem,
  deleteItem,
  draftFrom,
  emptyDraft,
  getItem,
  listItems,
  messageForContent,
  setStatus,
  updateItem,
  validateDraft,
  type AdminItem,
  type ContentKind
} from "./admin-content.ts";

type Call = { url: string; init: RequestInit };

function fakeFetch(respond: (call: Call) => Response | Promise<Response>) {
  const calls: Call[] = [];
  const impl = async (input: RequestInfo | URL, init?: RequestInit) => {
    const call = { url: String(input), init: init ?? {} };
    calls.push(call);
    return respond(call);
  };
  return { fetch: impl as typeof fetch, calls };
}

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
const errorBody = (code: string) => ({ error: { code, message: "<b>pesan server</b>" } });
const headers = (call: Call) => call.init.headers as Record<string, string>;
const CSRF = "c".repeat(43);

// What the Go server actually sends: zero values are omitted (omitempty).
const wireAnnouncement = {
  id: 4,
  title: "Libur",
  status: "draft",
  publishedAt: null,
  createdAt: "2026-11-13T03:00:00.123456Z",
  updatedAt: "2026-11-13T03:00:00.123456Z"
};
const wireMaterial = {
  id: 9,
  topicId: 2,
  title: "Graf",
  description: "Simpul.",
  status: "published",
  sortOrder: 3,
  publishedAt: "2026-11-13T04:00:00Z",
  createdAt: "2026-11-13T03:00:00Z",
  updatedAt: "2026-11-13T04:00:00.5Z"
};
const meta = { page: 1, limit: 20, total: 1, totalPages: 1 };

async function caught(promise: Promise<unknown>): Promise<AdminContentError> {
  try {
    await promise;
  } catch (error) {
    assert.ok(error instanceof AdminContentError, `expected AdminContentError, got ${String(error)}`);
    return error;
  }
  assert.fail("expected the call to reject");
}

describe("reading", () => {
  it("lists announcements with defaults filled in for omitted fields", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, { data: [wireAnnouncement], meta }));

    const result = await listItems("announcement", { page: 1, limit: 20 }, undefined, { fetch });

    assert.equal(calls[0].url, "/api/v1/admin/announcements?page=1&limit=20");
    assert.equal(calls[0].init.method, "GET");
    assert.equal(calls[0].init.credentials, "same-origin");
    assert.equal(calls[0].init.cache, "no-store");
    assert.deepEqual(result.meta, meta);
    assert.deepEqual(result.items[0], {
      kind: "announcement",
      id: 4,
      title: "Libur",
      text: "",
      status: "draft",
      pinned: false,
      topicId: null,
      sortOrder: 0,
      publishedAt: null,
      createdAt: wireAnnouncement.createdAt,
      updatedAt: wireAnnouncement.updatedAt
    } satisfies AdminItem);
  });

  it("maps material description to text and keeps topicId and sortOrder", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, { data: wireMaterial }));

    const item = await getItem("material", 9, undefined, { fetch });

    assert.equal(calls[0].url, "/api/v1/admin/materials/9");
    assert.equal(item.text, "Simpul.");
    assert.equal(item.topicId, 2);
    assert.equal(item.sortOrder, 3);
    assert.equal(item.publishedAt, "2026-11-13T04:00:00Z");
  });

  it("keeps updatedAt exactly as sent, so the conflict check compares the same instant", async () => {
    const { fetch } = fakeFetch(() => json(200, { data: wireMaterial }));
    assert.equal((await getItem("material", 9, undefined, { fetch })).updatedAt, "2026-11-13T04:00:00.5Z");
  });

  const malformed: Array<[string, unknown]> = [
    ["no data", {}],
    ["string id", { data: { ...wireAnnouncement, id: "4" } }],
    ["unknown status", { data: { ...wireAnnouncement, status: "archived" } }],
    ["bad updatedAt", { data: { ...wireAnnouncement, updatedAt: "kemarin" } }],
    ["published without publishedAt", { data: { ...wireAnnouncement, status: "published" } }],
    ["negative sortOrder", { data: { ...wireMaterial, sortOrder: -1 } }]
  ];
  for (const [name, body] of malformed) {
    it(`rejects a 200 response with ${name}`, async () => {
      const { fetch } = fakeFetch(() => json(200, body));
      assert.equal((await caught(getItem("announcement", 4, undefined, { fetch }))).kind, "server");
    });
  }

  it("refuses an invalid id or page before calling the server", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, {}));
    for (const id of [0, -1, 1.5, Number.NaN]) {
      assert.equal((await caught(getItem("announcement", id, undefined, { fetch }))).kind, "invalid_request");
    }
    assert.equal((await caught(listItems("material", { page: 0, limit: 20 }, undefined, { fetch }))).kind, "invalid_request");
    assert.equal((await caught(listItems("material", { page: 1, limit: 101 }, undefined, { fetch }))).kind, "invalid_request");
    assert.equal(calls.length, 0);
  });

  it("lets an abort propagate untouched", async () => {
    const { fetch } = fakeFetch(() => {
      throw new DOMException("aborted", "AbortError");
    });
    await assert.rejects(listItems("announcement", { page: 1, limit: 20 }, undefined, { fetch }), (e: unknown) => e instanceof DOMException && e.name === "AbortError");
  });
});

describe("writing", () => {
  it("creates an announcement with only the fields that apply, plus CSRF", async () => {
    const { fetch, calls } = fakeFetch(() => json(201, { data: { ...wireAnnouncement, title: "Ujian" } }));
    const draft = { ...emptyDraft("announcement"), title: "  Ujian  ", text: "Besok jam 7.", pinned: true, topicId: 3, sortOrder: 5 };

    const item = await createItem("announcement", draft, CSRF, { fetch });

    assert.equal(item.title, "Ujian");
    assert.equal(calls[0].url, "/api/v1/admin/announcements");
    assert.equal(calls[0].init.method, "POST");
    assert.equal(headers(calls[0])["X-CSRF-Token"], CSRF);
    assert.equal(headers(calls[0])["Content-Type"], "application/json");
    assert.deepEqual(JSON.parse(String(calls[0].init.body)), { title: "Ujian", body: "Besok jam 7.", status: "draft", pinned: true });
  });

  it("creates a material with description, topic, and order, and no pinned flag", async () => {
    const { fetch, calls } = fakeFetch(() => json(201, { data: wireMaterial }));
    const draft = { ...emptyDraft("material"), title: "Graf", text: "Simpul.", topicId: 2, sortOrder: 3, pinned: true };

    await createItem("material", draft, CSRF, { fetch });

    assert.deepEqual(JSON.parse(String(calls[0].init.body)), { title: "Graf", description: "Simpul.", status: "draft", topicId: 2, sortOrder: 3 });
  });

  it("sends topicId null for a material in Umum", async () => {
    const { fetch, calls } = fakeFetch(() => json(201, { data: { ...wireMaterial, topicId: undefined } }));
    const item = await createItem("material", { ...emptyDraft("material"), title: "Panduan" }, CSRF, { fetch });
    assert.equal(JSON.parse(String(calls[0].init.body)).topicId, null);
    assert.equal(item.topicId, null);
  });

  it("updates with PATCH and the updatedAt it was given", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, { data: wireMaterial }));
    const draft = { ...emptyDraft("material"), title: "Graf", text: "Simpul.", topicId: 2, sortOrder: 3 };

    await updateItem("material", 9, draft, "2026-11-13T04:00:00.5Z", CSRF, { fetch });

    assert.equal(calls[0].url, "/api/v1/admin/materials/9");
    assert.equal(calls[0].init.method, "PATCH");
    assert.equal(JSON.parse(String(calls[0].init.body)).updatedAt, "2026-11-13T04:00:00.5Z");
  });

  it("publishes or unpublishes by resending the item with a new status", async () => {
    const current = draftFrom({ kind: "announcement", id: 4, title: "Libur", text: "Isi", status: "draft", pinned: true, topicId: null, sortOrder: 0, publishedAt: null, createdAt: wireAnnouncement.createdAt, updatedAt: wireAnnouncement.updatedAt });
    const { fetch, calls } = fakeFetch(() => json(200, { data: { ...wireAnnouncement, status: "published", publishedAt: "2026-11-13T05:00:00Z" } }));

    const item = await setStatus({ ...current, kind: "announcement", id: 4, updatedAt: wireAnnouncement.updatedAt }, "published", CSRF, { fetch });

    assert.equal(item.status, "published");
    assert.deepEqual(JSON.parse(String(calls[0].init.body)), { title: "Libur", body: "Isi", status: "published", pinned: true, updatedAt: wireAnnouncement.updatedAt });
  });

  it("deletes with DELETE and CSRF and accepts 204", async () => {
    const { fetch, calls } = fakeFetch(() => new Response(null, { status: 204 }));
    await deleteItem("announcement", 4, CSRF, { fetch });
    assert.equal(calls[0].url, "/api/v1/admin/announcements/4");
    assert.equal(calls[0].init.method, "DELETE");
    assert.equal(headers(calls[0])["X-CSRF-Token"], CSRF);
  });

  it("validates before sending and never calls the server with a bad draft", async () => {
    const { fetch, calls } = fakeFetch(() => json(201, {}));
    const error = await caught(createItem("announcement", { ...emptyDraft("announcement"), title: "   " }, CSRF, { fetch }));
    assert.equal(error.kind, "validation");
    assert.equal(calls.length, 0);
  });
});

describe("errors", () => {
  const mapping: Array<[number, string, AdminContentError["kind"]]> = [
    [401, "UNAUTHENTICATED", "unauthenticated"],
    [403, "FORBIDDEN", "forbidden"],
    [404, "NOT_FOUND", "not_found"],
    [409, "CONFLICT", "conflict"],
    [422, "VALIDATION_ERROR", "validation"],
    [413, "PAYLOAD_TOO_LARGE", "too_large"],
    [400, "INVALID_REQUEST", "invalid_request"],
    [500, "INTERNAL_ERROR", "server"],
    [502, "", "server"]
  ];
  for (const [status, code, kind] of mapping) {
    it(`maps ${status} ${code || "(no body)"} to ${kind} with fixed text`, async () => {
      const { fetch } = fakeFetch(() => (code ? json(status, errorBody(code)) : new Response("<html>Bad Gateway</html>", { status })));
      const error = await caught(updateItem("announcement", 4, { ...emptyDraft("announcement"), title: "x" }, wireAnnouncement.updatedAt, CSRF, { fetch }));
      assert.equal(error.kind, kind);
      assert.equal(error.message, messageForContent(kind));
      assert.ok(!error.message.includes("pesan server"), "server text must never be shown");
    });
  }

  it("reports network failures without the underlying text", async () => {
    const { fetch } = fakeFetch(() => {
      throw new TypeError("fetch failed: ECONNREFUSED");
    });
    const error = await caught(deleteItem("material", 9, CSRF, { fetch }));
    assert.equal(error.kind, "network");
    assert.ok(!error.message.includes("ECONNREFUSED"));
  });

  it("has an actionable Indonesian message for every kind", () => {
    for (const kind of ["unauthenticated", "forbidden", "not_found", "conflict", "validation", "too_large", "invalid_request", "network", "server"] as const) {
      const text = messageForContent(kind);
      assert.ok(text.length > 15, kind);
      assert.ok(!/error|exception|sql|stack/i.test(text), `${kind}: ${text}`);
    }
  });
});

describe("validateDraft", () => {
  const ok = (kind: ContentKind, patch: object) => validateDraft(kind, { ...emptyDraft(kind), title: "Judul", ...patch });

  it("accepts a normal draft", () => {
    assert.deepEqual(ok("announcement", {}), {});
    assert.deepEqual(ok("material", { topicId: 1, sortOrder: 2147483647 }), {});
  });

  it("requires a title after trimming and caps it at 160 characters, counted like the server", () => {
    assert.ok(ok("announcement", { title: " \n\t " }).title);
    assert.deepEqual(ok("announcement", { title: "é".repeat(160) }), {});
    assert.deepEqual(ok("announcement", { title: "😀".repeat(160) }), {}, "emoji count as one character each, as in Go");
    assert.ok(ok("announcement", { title: "a".repeat(161) }).title);
  });

  it("caps announcement body at 10000 and material description at 20000", () => {
    assert.deepEqual(ok("announcement", { text: "a".repeat(10000) }), {});
    assert.ok(ok("announcement", { text: "a".repeat(10001) }).text);
    assert.deepEqual(ok("material", { text: "a".repeat(20000) }), {});
    assert.ok(ok("material", { text: "a".repeat(20001) }).text);
    assert.equal(LIMITS.announcement.text, 10000);
    assert.equal(LIMITS.material.text, 20000);
  });

  it("requires a whole, non-negative 32-bit sort order for materials only", () => {
    for (const bad of [-1, 1.5, 2147483648, Number.NaN]) {
      assert.ok(ok("material", { sortOrder: bad }).sortOrder, String(bad));
    }
    assert.deepEqual(ok("announcement", { sortOrder: -1 }), {}, "announcements have no order");
  });

  it("rejects a topic id that is not a positive integer", () => {
    assert.ok(ok("material", { topicId: 0 }).topicId);
    assert.ok(ok("material", { topicId: 1.2 }).topicId);
  });
});
