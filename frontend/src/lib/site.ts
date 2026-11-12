// Nama pemilik situs. Satu-satunya sumber nama; ubah di sini bila perlu.
export const SITE_OWNER = "Fikri Rama Aulia";
export const SITE_TITLE = `${SITE_OWNER} — Web Pribadi`;
export const SITE_DESCRIPTION =
  "Web pribadi guru Informatika: pengumuman, materi, dan progres tugas kelompok kelas IX.";

export const NAV_ITEMS = [
  { href: "/#pengumuman", label: "Pengumuman" },
  { href: "/#materi", label: "Materi" },
  { href: "/progres", label: "Progres Tugas" }
] as const;
