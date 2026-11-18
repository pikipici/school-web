// Nama pemilik situs. Satu-satunya sumber nama; ubah di sini bila perlu.
export const SITE_OWNER = "Fikri Rama Aulia";
export const SITE_TITLE = `${SITE_OWNER} — Web Pribadi`;
export const SITE_DESCRIPTION =
  "Progres tugas kelompok Informatika kelas IX dan catatan perbaikan dari guru, terbuka tanpa login.";

export const NAV_ITEMS = [
  { href: "/", label: "Progres Tugas" },
  { href: "/pengumuman/", label: "Pengumuman" },
  { href: "/materi/", label: "Materi" }
] as const;
