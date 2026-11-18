import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  ProgressError,
  activateYear,
  createClass,
  createGroup,
  createYear,
  deleteClass,
  deleteRound,
  draftFromRound,
  fromWIBInput,
  getNotebook,
  getPublicProgress,
  listClasses,
  listGroups,
  listRounds,
  messageForProgress,
  putNotebook,
  recordRound,
  roundLabel,
  suggestEmail,
  suggestSubject,
  toWIBInput,
  updateGroup,
  validateClass,
  validateGroup,
  validateRound,
  validateYearLabel,
  type GroupDraft
} from "./progress.ts";

type Call = { url: string; init: RequestInit };

function fakeFetch(respond: (call: Call) => Response) {
  const calls: Call[] = [];
  const impl = async (input: RequestInfo | URL, init?: RequestInit) => {
    const call = { url: String(input), init: init ?? {} };
    calls.push(call);
    return respond(call);
  };
  return { fetch: impl as typeof fetch, calls };
}

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
const errorBody = (code: string) => ({ error: { code, message: "<b>pesan server</b>" } });
const headers = (c: Call) => c.init.headers as Record<string, string>;
const CSRF = "c".repeat(43);
const T = "2026-11-15T01:00:00.123456Z";

const wireRow = {
  id: 7,
  name: "Kelompok 1",
  classCode: "9A",
  topic: { code: "K1", title: "Struktur Data (Graf dan Tree)" },
  submission: "BELUM",
  review: "BELUM_DICEK",
  notebook: "BELUM_DITERIMA",
  complete: false,
  updatedAt: T
};
const wireClass = { id: 3, yearId: 1, code: "9A", name: "", active: true, groupCount: 2, createdAt: T, updatedAt: T };
const wireGroup = { id: 7, classId: 3, topicId: 1, name: "Kelompok 1", email: null, sortOrder: 0, active: true, createdAt: T, updatedAt: T };
const draft = (over: Partial<GroupDraft> = {}): GroupDraft => ({ classId: 3, topicId: 1, name: "Kelompok 1", email: "", sortOrder: 0, active: true, ...over });

describe("validation", () => {
  it("checks academic year labels like the server", () => {
    assert.equal(validateYearLabel(" 2027/2028 "), null);
    for (const bad of ["2027/2029", "2028/2027", "2027-2028", "27/28", ""]) assert.ok(validateYearLabel(bad), bad);
  });

  it("checks class code and name", () => {
    assert.deepEqual(validateClass({ yearId: 1, code: " 9A ", name: "", active: true }), {});
    assert.deepEqual(validateClass({ yearId: 1, code: "IX-B.2", name: "x".repeat(80), active: true }), {});
    for (const code of ["", "-9A", "9/A", "A".repeat(21)]) assert.ok(validateClass({ yearId: 1, code, name: "", active: true }).code, code);
    assert.ok(validateClass({ yearId: 1, code: "9A", name: "é".repeat(81), active: true }).name);
  });

  it("checks group fields, counting characters not bytes", () => {
    assert.deepEqual(validateGroup(draft({ name: "界".repeat(60), email: " Kelompok1@Gmail.com " })), {});
    assert.ok(validateGroup(draft({ name: "界".repeat(61) })).name);
    assert.ok(validateGroup(draft({ name: "  " })).name);
    assert.ok(validateGroup(draft({ email: "bukan email" })).email);
    assert.ok(validateGroup(draft({ email: "a@b" })).email);
    assert.ok(validateGroup(draft({ topicId: 0 })).topicId);
    assert.ok(validateGroup(draft({ sortOrder: -1 })).sortOrder);
    assert.ok(validateGroup(draft({ sortOrder: Number.NaN })).sortOrder);
  });

  it("suggests the juknis email", () => {
    assert.equal(suggestEmail("Kelompok 1", "9A", "2026/2027"), "kelompok1.9a.2627.informatika@gmail.com");
    assert.equal(suggestEmail("", "9A", "2026/2027"), "");
    assert.equal(suggestEmail("Kelompok 1", "9A", ""), "");
  });
});

describe("public progress", () => {
  it("sends filters and validates every row", async () => {
    const f = fakeFetch(() => json(200, { data: { year: "2026/2027", classes: [{ code: "9A", name: "" }], groups: [wireRow] } }));
    const p = await getPublicProgress({ class: "9A", q: "  kelompok  ", status: "BELUM" }, undefined, { fetch: f.fetch });
    assert.equal(p.groups[0].topic.code, "K1");
    const url = new URL(f.calls[0].url, "http://x");
    assert.equal(url.pathname, "/api/v1/public/progress");
    assert.equal(url.searchParams.get("class"), "9A");
    assert.equal(url.searchParams.get("q"), "kelompok");
    assert.equal(url.searchParams.get("status"), "BELUM");

    const none = fakeFetch(() => json(200, { data: { year: "", classes: [], groups: [] } }));
    await getPublicProgress({}, undefined, { fetch: none.fetch });
    assert.equal(none.calls[0].url, "/api/v1/public/progress");
  });

  it("rejects unknown status values and malformed rows", async () => {
    for (const row of [{ ...wireRow, submission: "SELESAI" }, { ...wireRow, topic: null }, { ...wireRow, updatedAt: "kemarin" }]) {
      const f = fakeFetch(() => json(200, { data: { year: "2026/2027", classes: [], groups: [row] } }));
      await assert.rejects(getPublicProgress({}, undefined, { fetch: f.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "server");
    }
  });

  it("maps a 400 to a fixed message", async () => {
    const f = fakeFetch(() => json(400, errorBody("VALIDATION_ERROR")));
    await assert.rejects(getPublicProgress({ q: "x" }, undefined, { fetch: f.fetch }), (e: unknown) => {
      assert.ok(e instanceof ProgressError && e.kind === "validation");
      assert.ok(!e.message.includes("pesan server"));
      return true;
    });
  });
});

describe("admin", () => {
  it("writes send CSRF and JSON; activate and delete expect 204", async () => {
    const f = fakeFetch((c) => {
      if (c.url.endsWith("/activate") || c.init.method === "DELETE") return new Response(null, { status: 204 });
      if (c.url.includes("academic-years")) return json(201, { data: { id: 2, label: "2027/2028", active: false, createdAt: T, updatedAt: T } });
      return json(201, { data: wireClass });
    });
    await createYear(" 2027/2028 ", CSRF, { fetch: f.fetch });
    await activateYear(2, CSRF, { fetch: f.fetch });
    await createClass({ yearId: 1, code: " 9A ", name: " Pagi ", active: true }, CSRF, { fetch: f.fetch });
    await deleteClass(3, CSRF, { fetch: f.fetch });
    for (const c of f.calls) assert.equal(headers(c)["X-CSRF-Token"], CSRF, c.url);
    assert.deepEqual(JSON.parse(String(f.calls[0].init.body)), { label: "2027/2028" });
    assert.equal(f.calls[1].init.body, undefined);
    assert.deepEqual(JSON.parse(String(f.calls[2].init.body)), { yearId: 1, code: "9A", name: "Pagi" });
  });

  it("validates before sending", async () => {
    const f = fakeFetch(() => json(201, { data: wireGroup }));
    await assert.rejects(createGroup(draft({ name: "" }), CSRF, { fetch: f.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "validation");
    await assert.rejects(createYear("2027/2029", CSRF, { fetch: f.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "validation");
    await assert.rejects(listGroups(0, undefined, { fetch: f.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "invalid_request");
    assert.equal(f.calls.length, 0);
  });

  it("update sends active and updatedAt unchanged; email lower-cased", async () => {
    const f = fakeFetch(() => json(200, { data: { ...wireGroup, email: "kelompok1@gmail.com" } }));
    const g = await updateGroup(7, draft({ email: " Kelompok1@Gmail.com ", active: false }), T, CSRF, { fetch: f.fetch });
    assert.equal(g.email, "kelompok1@gmail.com");
    assert.equal(f.calls[0].init.method, "PATCH");
    assert.deepEqual(JSON.parse(String(f.calls[0].init.body)), {
      classId: 3,
      topicId: 1,
      name: "Kelompok 1",
      email: "kelompok1@gmail.com",
      sortOrder: 0,
      active: false,
      updatedAt: T
    });
  });

  it("lists with the right query and validates items", async () => {
    const f = fakeFetch(() => json(200, { data: [wireClass] }));
    assert.equal((await listClasses(null, undefined, { fetch: f.fetch }))[0].groupCount, 2);
    await listClasses(5, undefined, { fetch: f.fetch });
    assert.equal(f.calls[0].url, "/api/v1/admin/classes");
    assert.equal(f.calls[1].url, "/api/v1/admin/classes?year=5");
    const bad = fakeFetch(() => json(200, { data: [{ ...wireGroup, email: 5 }] }));
    await assert.rejects(listGroups(3, undefined, { fetch: bad.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "server");
  });

  it("maps every server code", async () => {
    const cases: [number, string, string][] = [
      [401, "UNAUTHENTICATED", "unauthenticated"],
      [403, "FORBIDDEN", "forbidden"],
      [404, "NOT_FOUND", "not_found"],
      [409, "CONFLICT", "conflict"],
      [409, "REVIEW_RULE", "review_rule"],
      [409, "DUPLICATE", "duplicate"],
      [409, "IN_USE", "in_use"],
      [422, "INVALID_REFERENCE", "reference"],
      [422, "VALIDATION_ERROR", "validation"],
      [500, "INTERNAL_ERROR", "server"]
    ];
    for (const [status, code, kind] of cases) {
      const f = fakeFetch(() => json(status, errorBody(code)));
      await assert.rejects(deleteClass(3, CSRF, { fetch: f.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === kind, code);
    }
    const offline = (async () => {
      throw new TypeError("offline");
    }) as typeof fetch;
    await assert.rejects(listClasses(null, undefined, { fetch: offline }), (e: unknown) => e instanceof ProgressError && e.kind === "network");
    assert.match(messageForProgress("in_use"), /Nonaktifkan/);
    assert.match(messageForProgress("review_rule"), /Sedang Diperiksa/);
  });
});

describe("submission rounds", () => {
  const baseDraft = { receivedAt: "2026-11-15T08:30", senderEmail: "", subject: "", ppt: true, pdf: false, pptFilename: "", pdfFilename: "", note: "" };
  const now = new Date("2026-11-15T03:00:00Z"); // 10.00 WIB
  const wireRound = {
    id: 5,
    groupId: 7,
    roundNumber: 0,
    receivedAt: "2026-11-15T01:30:00Z",
    senderEmail: null,
    subject: "",
    ppt: true,
    pdf: false,
    pptFilename: "",
    pdfFilename: "",
    note: "",
    createdAt: T,
    updatedAt: T
  };

  it("converts WIB wall-clock time both ways", () => {
    assert.equal(toWIBInput("2026-11-15T01:30:00Z"), "2026-11-15T08:30");
    assert.equal(toWIBInput("2026-11-14T20:00:00Z"), "2026-11-15T03:00");
    assert.equal(fromWIBInput("2026-11-15T08:30"), "2026-11-15T08:30:00+07:00");
    assert.equal(Date.parse(fromWIBInput("2026-11-15T08:30") ?? ""), Date.parse("2026-11-15T01:30:00Z"));
    for (const bad of ["", "2026-11-15", "2026-02-30T08:00", "2026-11-15T25:00", "15/11/2026 08:30"]) assert.equal(fromWIBInput(bad), null, bad);
  });

  it("labels rounds and suggests the juknis subject", () => {
    assert.equal(roundLabel(0), "Pengumpulan awal");
    assert.equal(roundLabel(2), "Revisi 2");
    assert.equal(suggestSubject(0, "9A", "Kelompok 1", "Struktur Data (Graf dan Tree)"), "Proyek Informatika IX - 9A - Kelompok 1 - Struktur Data");
    assert.equal(suggestSubject(1, "9A", "Kelompok 1", "Blogspot"), "REVISI 1 - Proyek Informatika IX - 9A - Kelompok 1 - Blogspot");
  });

  it("validates like the server", () => {
    assert.deepEqual(validateRound(baseDraft, now), {});
    assert.deepEqual(validateRound({ ...baseDraft, receivedAt: "2026-11-15T10:04" }, now), {}, "small clock skew is fine");
    assert.ok(validateRound({ ...baseDraft, receivedAt: "2026-11-15T11:00" }, now).receivedAt, "future");
    assert.ok(validateRound({ ...baseDraft, receivedAt: "" }, now).receivedAt);
    assert.ok(validateRound({ ...baseDraft, ppt: false }, now).files);
    assert.ok(validateRound({ ...baseDraft, pdfFilename: "a.pdf" }, now).pdfFilename, "pdf name without pdf");
    assert.ok(validateRound({ ...baseDraft, pptFilename: "dir/a.pptx" }, now).pptFilename);
    assert.ok(validateRound({ ...baseDraft, senderEmail: "bukan" }, now).senderEmail);
    assert.ok(validateRound({ ...baseDraft, subject: "a\nb" }, now).subject);
    assert.ok(validateRound({ ...baseDraft, note: "é".repeat(2001) }, now).note);
  });

  it("records with the next number, WIB time as an instant, and CSRF", async () => {
    const f = fakeFetch(() => json(201, { data: wireRound }));
    const r = await recordRound(7, 0, { ...baseDraft, pdfFilename: "ignored" }, CSRF, { fetch: f.fetch }).catch((e) => e);
    // pdfFilename without pdf is a validation error: nothing is sent.
    assert.ok(r instanceof ProgressError && r.kind === "validation");
    assert.equal(f.calls.length, 0);

    await recordRound(7, 0, { ...baseDraft, receivedAt: "2026-11-14T08:30", senderEmail: " Kelompok1@Gmail.com ", pptFilename: " Graf.pptx " }, CSRF, { fetch: f.fetch });
    const call = f.calls[0];
    assert.equal(call.url, "/api/v1/admin/groups/7/submissions");
    assert.equal(headers(call)["X-CSRF-Token"], CSRF);
    assert.deepEqual(JSON.parse(String(call.init.body)), {
      roundNumber: 0,
      receivedAt: "2026-11-14T08:30:00+07:00",
      senderEmail: "kelompok1@gmail.com",
      subject: "",
      ppt: true,
      pdf: false,
      pptFilename: "Graf.pptx",
      pdfFilename: "",
      note: ""
    });
  });

  it("lists rounds with the computed lane and maps NOT_LAST_ROUND", async () => {
    const f = fakeFetch(() => json(200, { data: { submission: "SEBAGIAN", rounds: [wireRound] } }));
    const out = await listRounds(7, undefined, { fetch: f.fetch });
    assert.equal(out.submission, "SEBAGIAN");
    assert.equal(draftFromRound(out.rounds[0]).receivedAt, "2026-11-15T08:30");
    const bad = fakeFetch(() => json(200, { data: { submission: "SELESAI", rounds: [] } }));
    await assert.rejects(listRounds(7, undefined, { fetch: bad.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "server");
    const busy = fakeFetch(() => json(409, errorBody("NOT_LAST_ROUND")));
    await assert.rejects(deleteRound(5, CSRF, { fetch: busy.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "not_last");
  });

  it("notebook: rejects inconsistent replies; sends null time for 'now'", async () => {
    const f = fakeFetch(() => json(200, { data: { received: true, receivedAt: T, note: "", updatedAt: T } }));
    await putNotebook(7, { received: true, receivedAt: "", note: " biru " }, CSRF, { fetch: f.fetch });
    assert.equal(f.calls[0].init.method, "PUT");
    assert.deepEqual(JSON.parse(String(f.calls[0].init.body)), { received: true, receivedAt: null, note: "biru" });
    await putNotebook(7, { received: false, receivedAt: "2026-11-15T08:30", note: "" }, CSRF, { fetch: f.fetch });
    assert.deepEqual(JSON.parse(String(f.calls[1].init.body)), { received: false, receivedAt: null, note: "" });
    const odd = fakeFetch(() => json(200, { data: { received: true, receivedAt: null, note: "", updatedAt: null } }));
    await assert.rejects(getNotebook(7, undefined, { fetch: odd.fetch }), (e: unknown) => e instanceof ProgressError && e.kind === "server");
  });
});
