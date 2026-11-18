import assert from "node:assert/strict";
import { it } from "node:test";
import { filterGroups, groupHeadline } from "./public-progress.ts";
const base = { id: 1, name: "Kelompok Mawar", classCode: "IX A", topic: { code: "T1", title: "Graf" }, submission: "LENGKAP" as const, review: "DIPERIKSA" as const, notebook: "BELUM_DITERIMA" as const, complete: false, updatedAt: "2026-11-18T08:00:00Z" };
it("filters revision state independently of submission completeness", () => {
 const revision = { ...base, id: 2, review: "PERLU_REVISI" as const };
 assert.deepEqual(filterGroups([base, revision], "", "revision", ""), [revision]);
});
it("combines class, trimmed name search, and completion filters", () => {
 const complete = { ...base, complete: true };
 assert.deepEqual(filterGroups([base, complete], "ix a", "complete", "  MAWAR "), [complete]);
 assert.deepEqual(filterGroups([complete], "IX B", "all", ""), []);
});
it("prioritizes completion and revision without inventing progress percentages", () => {
 assert.equal(groupHeadline({ ...base, complete: true }).label, "Tuntas");
 assert.equal(groupHeadline({ ...base, review: "PERLU_REVISI" }).label, "Perlu perbaikan");
 assert.equal(groupHeadline({ ...base, submission: "BELUM" }).label, "Belum mengumpulkan");
 assert.equal(groupHeadline({ ...base, submission: "SEBAGIAN" }).label, "Lengkapi pengumpulan");
 assert.equal(groupHeadline(base).label, "Sedang diperiksa");
 assert.equal(groupHeadline({ ...base, review: "DISETUJUI" }).label, "Menunggu buku tulis");
 assert.equal(groupHeadline({ ...base, review: "REVISI_DITERIMA" }).label, "Revisi menunggu pemeriksaan");
 assert.equal(groupHeadline({ ...base, review: "BELUM_DICEK" }).label, "Menunggu pemeriksaan");
});
