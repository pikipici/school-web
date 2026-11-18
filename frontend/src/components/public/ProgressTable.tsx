"use client";

import { useEffect, useState } from "react";
import { NOTEBOOK_LABEL, REVIEW_LABEL, SUBMISSION_LABEL, getPublicProgress, type PublicProgress } from "@/lib/api-client/progress";
import { filterGroups, groupHeadline, type GroupFilter } from "@/lib/public-progress";
import { formatWIB } from "@/lib/format";
import { GroupFindings } from "./GroupFindings";

type Load = { status: "loading" } | { status: "error" } | { status: "ready"; data: PublicProgress };
const filters: { value: GroupFilter; label: string }[] = [
 { value: "all", label: "Semua" }, { value: "revision", label: "Perlu perbaikan" }, { value: "complete", label: "Tuntas" }
];

export function ProgressTable() {
 const [load, setLoad] = useState<Load>({ status: "loading" });
 const [tick, setTick] = useState(0);
 const [cls, setCls] = useState("");
 const [filter, setFilter] = useState<GroupFilter>("all");
 const [q, setQ] = useState("");
 useEffect(() => {
  const controller = new AbortController();
  getPublicProgress({}, controller.signal).then(
   data => { if (!controller.signal.aborted) setLoad({ status: "ready", data }); },
   () => { if (!controller.signal.aborted) setLoad({ status: "error" }); }
  );
  return () => controller.abort();
 }, [tick]);
 function reset() { setQ(""); setCls(""); setFilter("all"); }
 if (load.status === "loading") return <p className="public-state loading" role="status">Memuat progres kelompok…</p>;
 if (load.status === "error") return <div className="public-state error" role="alert"><p>Progres kelompok tidak dapat dimuat. Silakan coba lagi.</p><button className="retry-button" type="button" onClick={() => { setLoad({ status: "loading" }); setTick(n => n + 1); }}>Coba lagi</button></div>;
 const { data } = load;
 const all = data.groups;
 const rows = filterGroups(all, cls, filter, q);
 const scoped = filterGroups(all, cls, "all", "");
 const active = !!(cls || q || filter !== "all");
 if (!all.length) return <div className="empty-block progress-empty" role="status"><h3>Progres kelompok akan muncul di sini</h3><p>Guru belum mendaftarkan kelompok{data.year ? ` untuk tahun ajaran ${data.year}` : ""}. Setelah didaftarkan, kamu bisa melihat progres dan catatan perbaikan tanpa login.</p></div>;
 return <>
  <form className="group-finder" role="search" aria-label="Cari progres kelompok" onSubmit={e => e.preventDefault()}>
   <div className="finder-inputs">
    <div className="filter"><label htmlFor="cari-kelompok">Cari kelompokmu</label><input id="cari-kelompok" type="search" placeholder="Ketik nama kelompok…" maxLength={60} value={q} onChange={e => setQ(e.target.value)} /></div>
    <div className="filter"><label htmlFor="saring-kelas">Kelas</label><select id="saring-kelas" value={cls} onChange={e => setCls(e.target.value)}><option value="">Semua kelas</option>{data.classes.map(c => <option key={c.code} value={c.code}>{c.name ? `${c.code} — ${c.name}` : c.code}</option>)}</select></div>
   </div>
   <div className="finder-bottom"><div className="group-filter-buttons" role="group" aria-label="Saring status kelompok">{filters.map(f => <button key={f.value} type="button" aria-pressed={filter === f.value} onClick={() => setFilter(f.value)}>{f.label}</button>)}</div>{active ? <button type="button" className="clear-filters" onClick={reset}>Hapus filter</button> : null}</div>
  </form>
  <div className="progress-overview" aria-label="Ringkasan kelompok pada kelas yang dipilih"><p><strong>{scoped.length}</strong> kelompok</p><p><strong>{scoped.filter(g => g.review === "PERLU_REVISI").length}</strong> perlu perbaikan</p><p><strong>{scoped.filter(g => g.complete).length}</strong> tuntas</p></div>
  <div className="progress-results"><h2>Progres kelompok</h2><p role="status" aria-live="polite">{rows.length} kelompok ditampilkan{data.year ? ` · ${data.year}` : ""}</p></div>
  {!rows.length ? <div className="empty-block"><h3>Kelompok tidak ditemukan</h3><p>Coba nama lain atau hapus filter untuk melihat semua kelompok.</p><button type="button" className="button-secondary" onClick={reset}>Tampilkan semua kelompok</button></div> : <div className="group-progress-list">{rows.map(g => {
   const headline = groupHeadline(g);
   return <article className="group-progress" key={g.id} aria-labelledby={`kelompok-${g.id}`}>
    <header className="group-card-header"><div><p className="group-class">Kelas {g.classCode}</p><h3 id={`kelompok-${g.id}`}>{g.name}</h3></div><span className={`group-headline headline-${headline.tone}`}>{headline.label}</span></header>
    <p className="group-topic"><span>Topik tugas</span> {g.topic.code} · {g.topic.title}</p>
    <dl className="group-lanes">
     <div><dt>Berkas digital</dt><dd>{SUBMISSION_LABEL[g.submission]}</dd></div>
     <div><dt>Pemeriksaan</dt><dd>{REVIEW_LABEL[g.review]}</dd></div>
     <div><dt>Buku tulis</dt><dd>{NOTEBOOK_LABEL[g.notebook]}</dd></div>
    </dl>
    <GroupFindings groupId={g.id} />
    <p className="group-updated">Diperbarui <time dateTime={g.updatedAt}>{formatWIB(g.updatedAt)}</time></p>
   </article>;
  })}</div>}
 </>;
}
