export function activePublicNav(path: string): string | null {
 const normalized = path.replace(/\/+$/, "") || "/";
 if (normalized === "/" || normalized === "/progres") return "/";
 for (const section of ["/pengumuman", "/materi"]) {
  if (normalized === section || normalized.startsWith(`${section}/`)) return `${section}/`;
 }
 return null;
}

export function contentPreview(text: string): string {
 const normalized = text.replace(/\s+/g, " ").trim();
 const characters = Array.from(normalized);
 return normalized ? characters.length > 180 ? `${characters.slice(0, 180).join("").trimEnd()}…` : normalized : "Buka untuk membaca selengkapnya.";
}
