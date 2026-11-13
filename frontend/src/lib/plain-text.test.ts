import assert from "node:assert/strict";
import test from "node:test";

import { toPlainTextModel } from "./plain-text.ts";

test("blank lines form paragraphs and ordinary newlines remain text", () => {
  assert.deepEqual(toPlainTextModel("Baris satu\nbaris dua\n\nParagraf dua"), [
    [{ kind: "text", value: "Baris satu\nbaris dua" }],
    [{ kind: "text", value: "Paragraf dua" }]
  ]);
});

test("only absolute HTTP(S) URLs become links", () => {
  assert.deepEqual(toPlainTextModel("Buka https://contoh.id/a?q=1 dan http://example.com. Jangan javascript:alert(1) atau ftp://x.test"), [[
    { kind: "text", value: "Buka " },
    { kind: "link", value: "https://contoh.id/a?q=1", href: "https://contoh.id/a?q=1" },
    { kind: "text", value: " dan " },
    { kind: "link", value: "http://example.com", href: "http://example.com" },
    { kind: "text", value: ". Jangan javascript:alert(1) atau ftp://x.test" }
  ]]);
});

test("HTML is retained as visible plain text, never interpreted", () => {
  assert.deepEqual(toPlainTextModel('<img src=x onerror="alert(1)"> <script>bad()</script>'), [[
    { kind: "text", value: '<img src=x onerror="alert(1)"> <script>bad()</script>' }
  ]]);
});

test("normalizes CRLF and ignores whitespace-only paragraph separators", () => {
  assert.deepEqual(toPlainTextModel("A\r\n  \r\nB\n\n\nC"), [[{ kind: "text", value: "A" }], [{ kind: "text", value: "B" }], [{ kind: "text", value: "C" }]]);
});
