import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  ADMIN_LOGIN_PATH,
  ADMIN_LOGOUT_PATH,
  ADMIN_ME_PATH,
  AdminApiError,
  getSession,
  login,
  logout,
  messageFor
} from "./admin.ts";

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

const json = (status: number, body: unknown, headers: Record<string, string> = {}) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", ...headers }
  });

const okSession = {
  data: { username: "fikri", csrfToken: "c".repeat(43), expiresAt: "2026-11-13T20:00:00Z" }
};

const errorBody = (code: string, message = "pesan dari server") => ({ error: { code, message } });

async function caught(promise: Promise<unknown>): Promise<AdminApiError> {
  try {
    await promise;
  } catch (error) {
    assert.ok(error instanceof AdminApiError, `expected AdminApiError, got ${String(error)}`);
    return error;
  }
  assert.fail("expected the call to reject");
}

describe("login", () => {
  it("posts JSON credentials to the login path with same-origin cookies and returns the session", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, okSession));

    const session = await login({ username: "fikri", password: "rahasia-123456" }, { fetch });

    assert.deepEqual(session, okSession.data);
    assert.equal(calls.length, 1);
    const { url, init } = calls[0];
    assert.equal(url, ADMIN_LOGIN_PATH);
    assert.equal(init.method, "POST");
    assert.equal(init.credentials, "same-origin");
    assert.equal(init.cache, "no-store");
    assert.equal((init.headers as Record<string, string>)["Content-Type"], "application/json");
    assert.deepEqual(JSON.parse(String(init.body)), { username: "fikri", password: "rahasia-123456" });
    assert.ok(!url.includes("?") && !url.includes("rahasia"), "credentials must never appear in the URL");
  });

  const mapping: Array<[number, string, AdminApiError["kind"]]> = [
    [401, "INVALID_CREDENTIALS", "invalid_credentials"],
    [403, "FORBIDDEN", "forbidden"],
    [400, "INVALID_REQUEST", "invalid_request"],
    [413, "PAYLOAD_TOO_LARGE", "invalid_request"],
    [415, "UNSUPPORTED_MEDIA_TYPE", "invalid_request"],
    [503, "SERVICE_BUSY", "busy"],
    [500, "INTERNAL", "server"],
    [418, "SOMETHING_NEW", "server"]
  ];
  for (const [status, code, kind] of mapping) {
    it(`maps ${status} ${code} to "${kind}" with a fixed, safe message`, async () => {
      const { fetch } = fakeFetch(() => json(status, errorBody(code, "<script>alert(1)</script> kata sandi: rahasia")));

      const error = await caught(login({ username: "fikri", password: "rahasia-123456" }, { fetch }));

      assert.equal(error.kind, kind);
      assert.equal(error.message, messageFor(kind));
      assert.ok(!error.message.includes("script") && !error.message.includes("rahasia"), "server text must not be echoed");
    });
  }

  it("reports the wait time for rate limiting from Retry-After", async () => {
    const { fetch } = fakeFetch(() => json(429, errorBody("RATE_LIMITED"), { "Retry-After": "120" }));

    const error = await caught(login({ username: "fikri", password: "x".repeat(12) }, { fetch }));

    assert.equal(error.kind, "rate_limited");
    assert.equal(error.retryAfterSeconds, 120);
  });

  for (const header of [undefined, "abc", "-5", "0", "99999999", "1.5"]) {
    it(`falls back to a default wait when Retry-After is ${JSON.stringify(header)}`, async () => {
      const headers: Record<string, string> = header === undefined ? {} : { "Retry-After": header };
      const { fetch } = fakeFetch(() => json(429, errorBody("RATE_LIMITED"), headers));

      const error = await caught(login({ username: "fikri", password: "x".repeat(12) }, { fetch }));

      assert.equal(error.retryAfterSeconds, 60);
    });
  }

  it("reports a network failure without leaking the underlying error", async () => {
    const { fetch } = fakeFetch(() => {
      throw new TypeError("fetch failed: connect ECONNREFUSED 127.0.0.1:8300");
    });

    const error = await caught(login({ username: "fikri", password: "x".repeat(12) }, { fetch }));

    assert.equal(error.kind, "network");
    assert.ok(!error.message.includes("ECONNREFUSED"));
  });

  it("treats a non-JSON error page as a server error", async () => {
    const { fetch } = fakeFetch(() => new Response("<html>Bad Gateway</html>", { status: 502 }));

    const error = await caught(login({ username: "fikri", password: "x".repeat(12) }, { fetch }));

    assert.equal(error.kind, "server");
  });

  const malformed: Array<[string, unknown]> = [
    ["missing data", {}],
    ["missing csrfToken", { data: { username: "fikri", expiresAt: "2026-11-13T20:00:00Z" } }],
    ["empty username", { data: { username: "", csrfToken: "c", expiresAt: "2026-11-13T20:00:00Z" } }],
    ["bad expiresAt", { data: { username: "fikri", csrfToken: "c", expiresAt: "kapan-kapan" } }],
    ["wrong types", { data: { username: 1, csrfToken: 2, expiresAt: 3 } }],
    ["array body", []]
  ];
  for (const [name, body] of malformed) {
    it(`rejects a 200 response with ${name}`, async () => {
      const { fetch } = fakeFetch(() => json(200, body));

      const error = await caught(login({ username: "fikri", password: "x".repeat(12) }, { fetch }));

      assert.equal(error.kind, "server");
    });
  }
});

describe("getSession", () => {
  it("returns the session on 200 using a credentialed, uncached GET", async () => {
    const { fetch, calls } = fakeFetch(() => json(200, okSession));

    const session = await getSession(undefined, { fetch });

    assert.deepEqual(session, okSession.data);
    assert.equal(calls[0].url, ADMIN_ME_PATH);
    assert.equal(calls[0].init.method, "GET");
    assert.equal(calls[0].init.credentials, "same-origin");
    assert.equal(calls[0].init.cache, "no-store");
  });

  it("returns null when not logged in", async () => {
    const { fetch } = fakeFetch(() => json(401, errorBody("UNAUTHENTICATED")));

    assert.equal(await getSession(undefined, { fetch }), null);
  });

  it("throws for server errors instead of pretending to be logged out", async () => {
    const { fetch } = fakeFetch(() => json(500, errorBody("INTERNAL")));

    assert.equal((await caught(getSession(undefined, { fetch }))).kind, "server");
  });

  it("throws a network error when the server is unreachable", async () => {
    const { fetch } = fakeFetch(() => {
      throw new TypeError("fetch failed");
    });

    assert.equal((await caught(getSession(undefined, { fetch }))).kind, "network");
  });

  it("lets an abort propagate untouched so callers can ignore it", async () => {
    const controller = new AbortController();
    const { fetch } = fakeFetch(({ init }) => {
      if (init.signal?.aborted) {
        throw new DOMException("aborted", "AbortError");
      }
      return json(200, okSession);
    });
    controller.abort();

    await assert.rejects(getSession(controller.signal, { fetch }), (error: unknown) => {
      return error instanceof DOMException && error.name === "AbortError";
    });
  });
});

describe("logout", () => {
  it("posts with the CSRF token header and same-origin cookies", async () => {
    const { fetch, calls } = fakeFetch(() => new Response(null, { status: 204 }));

    await logout("t".repeat(43), { fetch });

    assert.equal(calls[0].url, ADMIN_LOGOUT_PATH);
    assert.equal(calls[0].init.method, "POST");
    assert.equal(calls[0].init.credentials, "same-origin");
    assert.equal((calls[0].init.headers as Record<string, string>)["X-CSRF-Token"], "t".repeat(43));
  });

  it("surfaces a rejected logout so the UI does not claim success", async () => {
    const { fetch } = fakeFetch(() => json(403, errorBody("FORBIDDEN")));

    assert.equal((await caught(logout("bad", { fetch }))).kind, "forbidden");
  });

  it("reports network failures", async () => {
    const { fetch } = fakeFetch(() => {
      throw new TypeError("fetch failed");
    });

    assert.equal((await caught(logout("t".repeat(43), { fetch }))).kind, "network");
  });
});

describe("messageFor", () => {
  it("has an actionable Indonesian message for every kind", () => {
    const kinds: AdminApiError["kind"][] = [
      "invalid_credentials",
      "rate_limited",
      "busy",
      "forbidden",
      "invalid_request",
      "network",
      "server"
    ];
    for (const kind of kinds) {
      const text = messageFor(kind);
      assert.ok(text.length > 10, `${kind} message too short`);
      assert.ok(!/error|exception|stack|sql/i.test(text), `${kind} message leaks technical wording: ${text}`);
    }
  });
});
