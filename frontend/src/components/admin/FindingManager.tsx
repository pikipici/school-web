"use client";

import { useEffect, useState, type FormEvent } from "react";
import { FindingError, deleteFinding, getReview, listFindings, publishFindings, resolveFinding, saveFinding, setReview, validateFinding, type Finding, type FindingDraft, type ReviewRecord } from "@/lib/api-client/findings";
import { REVIEW_LABEL, type Review } from "@/lib/api-client/progress";

const blank:FindingDraft={fileType:"PPT",location:"",problem:"",action:"",priority:"WAJIB"};
export function FindingManager({groupId,csrf,onExpired}:{groupId:number;csrf:string;onExpired:()=>void}){
 const [data,setData]=useState<{rows:Finding[];review:ReviewRecord}|null>(null);
 const [draft,setDraft]=useState<FindingDraft>(blank);
 const [edit,setEdit]=useState<Finding|null>(null);
 const [status,setStatus]=useState<Review>("DIPERIKSA");
 const [notice,setNotice]=useState<{error:boolean;text:string}|null>(null);
 const [busy,setBusy]=useState(false);
 const [ready,setReady]=useState(false);
 const [tick,setTick]=useState(0);
 const [confirm,setConfirm]=useState<"publish"|number|null>(null);
 const locked=busy||!ready;
 const unsaved=edit!==null||(Object.keys(blank) as (keyof FindingDraft)[]).some(key=>draft[key]!==blank[key]);
 const publicationBlocked=locked||unsaved||!data?.rows.some(f=>!f.publishedAt);
 function handleError(e:unknown){if(e instanceof FindingError&&e.kind==="unauthenticated")onExpired();setNotice({error:true,text:e instanceof FindingError?e.message:"Catatan tidak dapat diproses. Coba lagi."});}
 useEffect(()=>{
  const c=new AbortController();
  Promise.all([listFindings(groupId,c.signal),getReview(groupId,c.signal)]).then(([rows,review])=>{
   if(!c.signal.aborted){setData({rows,review});setStatus(review.status);setReady(true);setNotice(null);setConfirm(null);setEdit(null);setDraft(blank);}
  },e=>{if(!c.signal.aborted){setReady(false);if(e instanceof FindingError&&e.kind==="unauthenticated")onExpired();setNotice({error:true,text:e instanceof FindingError?e.message:"Catatan tidak dapat dimuat."});}});
  return()=>c.abort();
 },[groupId,tick,onExpired]);
 async function run(action:()=>Promise<unknown>,text:string){
  if(locked)return;
  setBusy(true);setReady(false);setNotice(null);setConfirm(null);
  let mutationFailed=false;
  try{
   try{await action();setNotice({error:false,text});}catch(e){mutationFailed=true;handleError(e);}
   const [rows,review]=await Promise.all([listFindings(groupId),getReview(groupId)]);
   setData({rows,review});setStatus(review.status);setReady(true);
   if(edit&&!rows.some(f=>f.id===edit.id&&!f.publishedAt&&f.updatedAt===edit.updatedAt)){
    setEdit(null);setDraft(blank);
    if(mutationFailed)setNotice({error:true,text:"Draf berubah, terbit, atau dihapus. Data terbaru dimuat; pilih ulang draf sebelum mengubah."});
   }
  }catch(e){handleError(e);setReady(false);}finally{setBusy(false);}
 }
 function save(e:FormEvent){e.preventDefault();if(locked)return;const error=validateFinding(draft);if(error){setNotice({error:true,text:error});return;}void run(async()=>{await saveFinding(groupId,draft,csrf,edit);setDraft(blank);setEdit(null);},"Draf catatan disimpan. Belum terlihat publik.");}
 return <section className="content-section" aria-labelledby="catatan-admin"><h2 id="catatan-admin">Catatan perbaikan</h2>
  <p>Catat jenis tugas, letak kesalahan, dan apa yang harus diperbaiki. Draf hanya terlihat admin. Catatan terbit muncul di bawah progres kelompok pada halaman depan.</p>
  {notice?<p role={notice.error?"alert":"status"} className={notice.error?"field-error":""}>{notice.text}</p>:null}
  {busy?<p role="status">Menyimpan dan memuat data catatan terbaru…</p>:!ready?<><p role="status">Catatan belum siap. Tindakan dinonaktifkan sampai data terbaru berhasil dimuat. Muat ulang akan mengosongkan formulir yang belum disimpan.</p><button type="button" className="button-secondary" onClick={()=>{setReady(false);setTick(n=>n+1);}}>Muat ulang catatan</button></>:null}
  {unsaved?<p role="status">Simpan atau batalkan perubahan formulir sebelum menerbitkan catatan.</p>:null}
  {data?<> 
   <div className="form-field"><label htmlFor="status-pemeriksaan">Status pemeriksaan</label><select id="status-pemeriksaan" disabled={locked} value={status} onChange={e=>setStatus(e.target.value as Review)}>{(Object.entries(REVIEW_LABEL) as [Review,string][]).map(([value,label])=><option key={value} value={value}>{label}</option>)}</select></div>
   <p>Status saat ini: <strong>{REVIEW_LABEL[data.review.status]}</strong>.</p>
   <div className="admin-actions"><button type="button" className="button-secondary" disabled={locked||status===data.review.status} onClick={()=>void run(()=>setReview(groupId,status,data.review.updatedAt,csrf),"Status pemeriksaan disimpan.")}>Simpan status</button>
    <button type="button" className="button-primary" disabled={publicationBlocked} onClick={()=>{if(!publicationBlocked)setConfirm("publish");}}>Terbitkan catatan dan status</button></div>
   {confirm==="publish"?<div className="confirmation" role="group" aria-label="Konfirmasi publikasi"><p>Semua draf akan terlihat siapa pun tanpa login, dengan status <strong>{REVIEW_LABEL[status]}</strong>. Pastikan tidak ada data pribadi dalam catatan.</p><button type="button" className="button-primary" disabled={publicationBlocked} onClick={()=>{if(!publicationBlocked)void run(()=>publishFindings(groupId,status,data.review.updatedAt,csrf),"Catatan dan status diterbitkan. Siswa dapat membacanya di halaman depan.");}}>Ya, terbitkan ke publik</button> <button type="button" className="button-secondary" disabled={busy} onClick={()=>setConfirm(null)}>Batal</button></div>:null}
   <ul className="finding-list">{data.rows.map(f=><li key={f.id}><p><strong>{f.fileType} — {f.location}</strong> · {f.publishedAt?"Terbit":"Draf"} · {f.resolved?"✓ Selesai":f.priority==="WAJIB"?"! Wajib diperbaiki":"○ Saran"}</p><p><strong>Masalah:</strong> {f.problem}</p><p><strong>Perbaikan:</strong> {f.action}</p>
    <div className="admin-actions">{!f.publishedAt?<><button type="button" className="button-secondary" aria-label={`Ubah draf ${f.fileType} — ${f.location}`} disabled={locked} onClick={()=>{setConfirm(null);setEdit(f);setDraft({fileType:f.fileType,location:f.location,problem:f.problem,action:f.action,priority:f.priority});}}>Ubah draf</button><button type="button" className="button-danger" aria-label={`Hapus draf ${f.fileType} — ${f.location}`} disabled={locked} onClick={()=>setConfirm(f.id)}>Hapus draf</button></>:!f.resolved?<button type="button" className="button-secondary" aria-label={`Tandai selesai ${f.fileType} — ${f.location}`} disabled={locked} onClick={()=>void run(()=>resolveFinding(f,csrf),"Catatan ditandai selesai.")}>Tandai selesai</button>:null}</div>
    {confirm===f.id?<div role="group" aria-label={`Konfirmasi hapus ${f.fileType} — ${f.location}`}><p>Hapus draf {f.fileType} — {f.location}?</p><button type="button" className="button-danger" aria-label={`Ya, hapus draf ${f.fileType} — ${f.location}`} disabled={locked} onClick={()=>void run(async()=>{await deleteFinding(f,csrf);if(edit?.id===f.id){setEdit(null);setDraft(blank);}},"Draf dihapus.")}>Ya, hapus</button> <button type="button" className="button-secondary" onClick={()=>setConfirm(null)}>Batal</button></div>:null}
   </li>)}</ul>
  </>:null}
  <form className="content-form" onSubmit={save}><fieldset disabled={locked||!data}><legend>{edit?"Ubah draf":"Tambah catatan (draf)"}</legend>
   <div className="form-field"><label htmlFor="temuan-jenis">Jenis tugas</label><select id="temuan-jenis" value={draft.fileType} onChange={e=>setDraft({...draft,fileType:e.target.value as FindingDraft["fileType"]})}>{["PPT","PDF","BUKU","PRESENTASI"].map(v=><option key={v}>{v}</option>)}</select></div>
   <div className="form-field"><label htmlFor="temuan-lokasi">Letak kesalahan (wajib, misalnya Slide 5)</label><input id="temuan-lokasi" required maxLength={200} value={draft.location} onChange={e=>setDraft({...draft,location:e.target.value})}/></div>
   <div className="form-field"><label htmlFor="temuan-masalah">Masalah (wajib)</label><input id="temuan-masalah" required maxLength={2000} value={draft.problem} onChange={e=>setDraft({...draft,problem:e.target.value})}/></div>
   <div className="form-field"><label htmlFor="temuan-perbaikan">Perbaikan yang diminta (wajib)</label><input id="temuan-perbaikan" required maxLength={2000} value={draft.action} onChange={e=>setDraft({...draft,action:e.target.value})}/></div>
   <div className="form-field"><label htmlFor="temuan-prioritas">Prioritas</label><select id="temuan-prioritas" value={draft.priority} onChange={e=>setDraft({...draft,priority:e.target.value as FindingDraft["priority"]})}><option value="WAJIB">Wajib diperbaiki</option><option value="SARAN">Saran</option></select></div>
   <button className="button-primary" type="submit">Simpan draf</button> {unsaved?<button className="button-secondary" type="button" onClick={()=>{setEdit(null);setDraft(blank);setConfirm(null);}}>{edit?"Batal ubah":"Batalkan draf baru"}</button>:null}
  </fieldset></form>
 </section>;
}
