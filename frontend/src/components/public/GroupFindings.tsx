"use client";

import { useEffect, useState } from "react";
import { getPublicFindings, type PublicFinding } from "@/lib/api-client/findings";

export function GroupFindings({ groupId }: { groupId: number }) {
 const [rows,setRows]=useState<PublicFinding[]|null>(null);
 const [error,setError]=useState(false);
 const [retry,setRetry]=useState(0);
 useEffect(()=>{
  const c=new AbortController();
  getPublicFindings(groupId,c.signal).then(data=>{if(!c.signal.aborted)setRows(data);},()=>{if(!c.signal.aborted)setError(true);});
  return ()=>c.abort();
 },[groupId,retry]);
 return <section className="group-findings" aria-label="Catatan perbaikan">
  <h4>Catatan perbaikan</h4>
  {error?<div role="alert"><p>Catatan belum dapat dimuat. Ini bukan berarti tidak ada perbaikan.</p><button type="button" className="button-secondary" onClick={()=>{setError(false);setRows(null);setRetry(n=>n+1);}}>Muat ulang catatan</button></div>
   :rows===null?<p role="status">Memuat catatan…</p>
   :rows.length===0?<p>Belum ada catatan perbaikan yang diterbitkan guru.</p>
   :<ul className="finding-list">{rows.map(f=><li key={f.id}>
    <p><strong>{f.fileType} — {f.location}</strong> <span className={`lane-chip ${f.resolved?"lane-done":f.priority==="WAJIB"?"lane-attention":"lane-none"}`}>{f.resolved?"✓ Selesai":f.priority==="WAJIB"?"! Wajib diperbaiki":"○ Saran"}</span></p>
    <p><strong>Masalah:</strong> {f.problem}</p><p><strong>Perbaikan:</strong> {f.action}</p>
   </li>)}</ul>}
 </section>;
}
