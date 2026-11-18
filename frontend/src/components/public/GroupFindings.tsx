"use client";

import { useEffect, useState } from "react";
import { getPublicFindings, type PublicFinding } from "@/lib/api-client/findings";

function FindingList({ rows }: { rows: PublicFinding[] }) {
 return <ul className="finding-list">{rows.map(f => <li key={f.id}>
  <div className="finding-heading"><strong>{f.fileType} — {f.location}</strong><span className={`lane-chip ${f.resolved ? "lane-done" : f.priority === "WAJIB" ? "lane-attention" : "lane-none"}`}>{f.resolved ? "✓ Selesai" : f.priority === "WAJIB" ? "Wajib diperbaiki" : "Saran"}</span></div>
  <p className="finding-problem"><strong>Masalah:</strong> {f.problem}</p><p className="finding-action"><strong>Yang perlu dilakukan:</strong> {f.action}</p>
 </li>)}</ul>;
}

export function GroupFindings({ groupId }: { groupId: number }) {
 const [rows, setRows] = useState<PublicFinding[] | null>(null);
 const [error, setError] = useState(false);
 const [retry, setRetry] = useState(0);
 useEffect(() => {
  const c = new AbortController();
  getPublicFindings(groupId, c.signal).then(data => { if (!c.signal.aborted) setRows(data); }, () => { if (!c.signal.aborted) setError(true); });
  return () => c.abort();
 }, [groupId, retry]);
 const open = rows?.filter(f => !f.resolved) ?? [];
 const resolved = rows?.filter(f => f.resolved) ?? [];
 return <section className="group-findings" aria-label="Catatan perbaikan">
  <h4>Catatan perbaikan</h4>
  {error ? <div role="alert"><p>Catatan belum dapat dimuat. Ini bukan berarti tidak ada perbaikan.</p><button type="button" className="button-secondary" onClick={() => { setError(false); setRows(null); setRetry(n => n + 1); }}>Muat ulang catatan</button></div>
   : rows === null ? <p role="status">Memuat catatan…</p>
   : rows.length === 0 ? <p className="finding-empty">Belum ada catatan yang diterbitkan. Tetap perhatikan status pemeriksaan di atas.</p>
   : <>{open.length ? <FindingList rows={open} /> : <p className="finding-empty">Semua catatan yang diterbitkan sudah ditandai selesai.</p>}{resolved.length ? <details className="resolved-findings"><summary>Catatan selesai ({resolved.length})</summary><FindingList rows={resolved} /></details> : null}</>}
 </section>;
}
