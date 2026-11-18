import assert from "node:assert/strict";
import { it } from "node:test";
import { activePublicNav, contentPreview } from "./public-content.ts";
it("marks exact public sections and their detail pages without matching unrelated prefixes", () => {
 for (const path of ["/", "/progres/", "/progres"]) assert.equal(activePublicNav(path), "/");
 assert.equal(activePublicNav("/pengumuman/baca/"), "/pengumuman/");
 assert.equal(activePublicNav("/materi/baca/"), "/materi/");
 assert.equal(activePublicNav("/materi"), "/materi/");
 assert.equal(activePublicNav("/material/"), null);
 assert.equal(activePublicNav("/admin/materi/ubah/"), null);
});
it("makes a bounded plain-text preview from nonempty lines", () => {
 assert.equal(contentPreview("\n  Pertama\r\n Kedua  "), "Pertama Kedua");
 assert.equal(contentPreview(""), "Buka untuk membaca selengkapnya.");
 assert.equal(contentPreview("a".repeat(200)).length, 181);
 assert.equal(contentPreview("<b>teks</b>"), "<b>teks</b>");
});
it("does not split an emoji at the preview boundary", () => {
 assert.equal(contentPreview("a".repeat(179) + "😀" + "more"), "a".repeat(179) + "😀…");
 assert.equal(contentPreview("😀".repeat(180)), "😀".repeat(180));
});
