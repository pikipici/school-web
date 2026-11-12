// Nama sementara. Ganti di sini saat nama asli diberikan; satu-satunya sumber nama situs.
export const SITE_OWNER = "Guru Informatika";
export const SITE_TITLE = `${SITE_OWNER} — Web Pribadi`;
export const SITE_DESCRIPTION =
  "Web pribadi guru Informatika: pengumuman, materi, dan progres tugas kelompok kelas IX.";

export const NAV_ITEMS = [
  { href: "/#pengumuman", label: "Pengumuman" },
  { href: "/#materi", label: "Materi" },
  { href: "/progres", label: "Progres Tugas" }
] as const;
