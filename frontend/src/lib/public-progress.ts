import type { PublicGroup } from "./api-client/progress";

export type GroupFilter = "all" | "revision" | "complete";

export function filterGroups(groups: PublicGroup[], cls: string, filter: GroupFilter, query: string): PublicGroup[] {
 const q = query.trim().toLocaleLowerCase("id");
 return groups.filter(g => (!cls || g.classCode.toLowerCase() === cls.toLowerCase())
  && (filter !== "revision" || g.review === "PERLU_REVISI")
  && (filter !== "complete" || g.complete)
  && (!q || g.name.toLocaleLowerCase("id").includes(q)));
}

export function groupHeadline(g: PublicGroup): { label: string; tone: string } {
 if (g.complete) return { label: "Tuntas", tone: "done" };
 if (g.review === "PERLU_REVISI") return { label: "Perlu perbaikan", tone: "attention" };
 if (g.submission === "BELUM") return { label: "Belum mengumpulkan", tone: "neutral" };
 if (g.submission === "SEBAGIAN") return { label: "Lengkapi pengumpulan", tone: "attention" };
 if (g.review === "DISETUJUI") return { label: "Menunggu buku tulis", tone: "neutral" };
 if (g.review === "DIPERIKSA") return { label: "Sedang diperiksa", tone: "neutral" };
 if (g.review === "REVISI_DITERIMA") return { label: "Revisi menunggu pemeriksaan", tone: "neutral" };
 return { label: "Menunggu pemeriksaan", tone: "neutral" };
}
