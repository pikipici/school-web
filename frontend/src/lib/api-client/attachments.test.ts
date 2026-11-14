import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  ATTACHMENT_LIMITS,
  AttachmentError,
  checkFile,
  deleteAttachment,
  downloadURL,
  formatSize,
  getPublicAttachments,
  listAdminAttachments,
  messageForAttachment,
  uploadAttachment,
  type UploadRequest
} from "./attachments.ts";

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
const CSRF = "c".repeat(43);

const wireAdmin = { id: 3, materialId: 9, name: "Graf.pdf", kind: "pdf", contentType: "application/pdf", size: 1200, createdAt: "2026-11-14T03:00:00.5Z" };
const pdfFile = (size = 10, name = "Materi K1.pdf") => new File([new Uint8Array(size)], name, { type: "application/pdf" });

function fakeTransport(status: number, body: unknown, progress: [number, number][] = []) {
  const calls: UploadRequest[] = [];
  const transport = async (req: UploadRequest) => {
    calls.push(req);
    for (const [a, b] of progress) req.onProgress?.(a, b);
    return { status, text: typeof body === "string" ? body : JSON.stringify(body) };
  };
  return { transport, calls };
}

describe("checkFile", () => {
  it("accepts pdf, ppt and pptx in any letter case up to 25 MiB", () => {
    for (const name of ["a.pdf", "B.PPT", "slide.Pptx", "dua.titik.pdf"]) assert.equal(checkFile({ name, size: 10 }), null, name);
    assert.equal(checkFile({ name: "a.pdf", size: ATTACHMENT_LIMITS.maxFile }), null);
  });
  it("rejects other names, empty files and files over the limit", () => {
    for (const name of ["a.pptm", "a.ppsx", "a.docx", "a.pdf.exe", "pdf", "a"]) assert.equal(checkFile({ name, size: 10 }), "type", name);
    assert.equal(checkFile({ name: "a.pdf", size: 0 }), "empty");
    assert.equal(checkFile({ name: "a.pdf", size: ATTACHMENT_LIMITS.maxFile + 1 }), "too_large");
  });
});

describe("formatSize", () => {
  it("uses Indonesian decimals and binary units", () => {
    assert.equal(formatSize(12), "12 B");
    assert.equal(formatSize(2048), "2 KB");
    assert.equal(formatSize(1536 * 1024), "1,5 MB");
    assert.equal(formatSize(25 * 1024 * 1024), "25 MB");
  });
});

describe("uploadAttachment", () => {
  it("posts multipart with CSRF, reports progress, and parses the result", async () => {
    const t = fakeTransport(201, { data: wireAdmin }, [[5, 10], [10, 10]]);
    const seen: number[] = [];
    const item = await uploadAttachment(9, pdfFile(), CSRF, { onProgress: (s) => seen.push(s) }, { transport: t.transport });
    assert.deepEqual(item, { id: 3, materialId: 9, name: "Graf.pdf", kind: "pdf", size: 1200, createdAt: "2026-11-14T03:00:00.5Z" });
    assert.deepEqual(seen, [5, 10]);
    const req = t.calls[0];
    assert.equal(req.url, "/api/v1/admin/materials/9/attachments");
    assert.equal(req.headers["X-CSRF-Token"], CSRF);
    assert.equal(req.headers["Content-Type"], undefined, "browser must set the multipart boundary");
    const part = req.body.get("file");
    assert.ok(part instanceof File && part.name === "Materi K1.pdf");
    assert.deepEqual([...req.body.keys()], ["file"]);
  });

  it("checks the file before sending anything", async () => {
    const t = fakeTransport(201, { data: wireAdmin });
    await assert.rejects(uploadAttachment(9, pdfFile(10, "a.docx"), CSRF, {}, { transport: t.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "type");
    await assert.rejects(uploadAttachment(9, pdfFile(0), CSRF, {}, { transport: t.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "empty");
    await assert.rejects(uploadAttachment(0, pdfFile(), CSRF, {}, { transport: t.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "invalid_request");
    assert.equal(t.calls.length, 0);
  });

  it("maps every server error code to a fixed message", async () => {
    const cases: [number, string, string][] = [
      [401, "UNAUTHENTICATED", "unauthenticated"],
      [403, "FORBIDDEN", "forbidden"],
      [404, "NOT_FOUND", "not_found"],
      [409, "ATTACHMENT_LIMIT", "limit"],
      [413, "PAYLOAD_TOO_LARGE", "too_large"],
      [415, "UNSUPPORTED_FILE_TYPE", "type"],
      [415, "MACRO_NOT_ALLOWED", "macro"],
      [415, "UNSUPPORTED_MEDIA_TYPE", "invalid_request"],
      [400, "INVALID_FILE_NAME", "name"],
      [400, "INVALID_REQUEST", "invalid_request"],
      [503, "SERVICE_BUSY", "busy"],
      [507, "STORAGE_QUOTA_EXCEEDED", "quota"],
      [507, "INSUFFICIENT_STORAGE", "disk"],
      [500, "INTERNAL_ERROR", "server"]
    ];
    for (const [status, code, kind] of cases) {
      const t = fakeTransport(status, errorBody(code));
      await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: t.transport }), (e: unknown) => {
        assert.ok(e instanceof AttachmentError, code);
        assert.equal(e.kind, kind, code);
        assert.ok(!e.message.includes("pesan server"), "server text must not leak");
        return true;
      });
    }
  });

  it("falls back to the status when the body is not JSON (e.g. Caddy 413)", async () => {
    const t = fakeTransport(413, "<html>Request Entity Too Large</html>");
    await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: t.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "too_large");
    const g = fakeTransport(502, "Bad Gateway");
    await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: g.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "server");
  });

  it("rejects a malformed success and leaves cancel as AbortError", async () => {
    const bad = fakeTransport(201, { data: { ...wireAdmin, kind: "exe" } });
    await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: bad.transport }), (e: unknown) => e instanceof AttachmentError && e.kind === "server");
    const aborting = async () => {
      throw new DOMException("x", "AbortError");
    };
    await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: aborting }), (e: unknown) => e instanceof DOMException && e.name === "AbortError");
    const offline = async () => {
      throw new TypeError("network");
    };
    await assert.rejects(uploadAttachment(9, pdfFile(), CSRF, {}, { transport: offline }), (e: unknown) => e instanceof AttachmentError && e.kind === "network");
  });
});

describe("listAdminAttachments and deleteAttachment", () => {
  it("lists with credentials and validates every item", async () => {
    const f = fakeFetch(() => json(200, { data: [wireAdmin] }));
    const items = await listAdminAttachments(9, undefined, { fetch: f.fetch });
    assert.equal(items.length, 1);
    assert.equal(f.calls[0].url, "/api/v1/admin/materials/9/attachments");
    assert.equal(f.calls[0].init.credentials, "same-origin");
    const bad = fakeFetch(() => json(200, { data: [{ ...wireAdmin, size: 0 }] }));
    await assert.rejects(listAdminAttachments(9, undefined, { fetch: bad.fetch }), (e: unknown) => e instanceof AttachmentError && e.kind === "server");
    const gone = fakeFetch(() => json(401, errorBody("UNAUTHENTICATED")));
    await assert.rejects(listAdminAttachments(9, undefined, { fetch: gone.fetch }), (e: unknown) => e instanceof AttachmentError && e.kind === "unauthenticated");
  });

  it("deletes with CSRF and maps 404", async () => {
    const f = fakeFetch(() => new Response(null, { status: 204 }));
    await deleteAttachment(3, CSRF, { fetch: f.fetch });
    assert.equal(f.calls[0].url, "/api/v1/admin/attachments/3");
    assert.equal(f.calls[0].init.method, "DELETE");
    assert.equal((f.calls[0].init.headers as Record<string, string>)["X-CSRF-Token"], CSRF);
    const missing = fakeFetch(() => json(404, errorBody("NOT_FOUND")));
    await assert.rejects(deleteAttachment(3, CSRF, { fetch: missing.fetch }), (e: unknown) => e instanceof AttachmentError && e.kind === "not_found");
  });
});

describe("public attachments", () => {
  it("lists only id, name, kind and size, and builds the download URL", async () => {
    const f = fakeFetch(() => json(200, { data: [{ id: 3, name: "Graf.pdf", kind: "pdf", size: 1200 }] }));
    assert.deepEqual(await getPublicAttachments(9, undefined, { fetch: f.fetch }), [{ id: 3, name: "Graf.pdf", kind: "pdf", size: 1200 }]);
    assert.equal(f.calls[0].url, "/api/v1/public/materials/9/attachments");
    assert.equal(downloadURL(3), "/api/v1/public/attachments/3");
    assert.throws(() => downloadURL(-1), AttachmentError);
  });

  it("treats a draft material as not found", async () => {
    const f = fakeFetch(() => json(404, errorBody("NOT_FOUND")));
    await assert.rejects(getPublicAttachments(9, undefined, { fetch: f.fetch }), (e: unknown) => e instanceof AttachmentError && e.kind === "not_found");
  });
});

describe("messages", () => {
  it("every kind has fixed Indonesian text", () => {
    assert.match(messageForAttachment("limit"), /5 lampiran/);
    assert.match(messageForAttachment("too_large"), /25 MB/);
  });
});
