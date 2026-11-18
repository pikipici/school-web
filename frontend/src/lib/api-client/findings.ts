import type { Review } from "./progress.ts";

export interface FindingDraft { fileType: "PPT" | "PDF" | "BUKU" | "PRESENTASI"; location: string; problem: string; action: string; priority: "WAJIB" | "SARAN" }
export interface PublicFinding extends FindingDraft { id: number; resolved: boolean; publishedAt: string; updatedAt: string }
export interface Finding extends FindingDraft { id: number; groupId: number; resolved: boolean; publishedAt: string | null; createdAt: string; updatedAt: string }
export interface ReviewRecord { status: Review; updatedAt: string }
type Deps={fetch?:typeof fetch};
const record=(v:unknown):v is Record<string,unknown>=>typeof v==="object"&&v!==null&&!Array.isArray(v);
const date=(v:unknown):v is string=>typeof v==="string"&&!Number.isNaN(Date.parse(v));
const positive=(v:unknown):v is number=>typeof v==="number"&&Number.isSafeInteger(v)&&v>0;
const reviews:readonly string[]=["BELUM_DICEK","DIPERIKSA","PERLU_REVISI","REVISI_DITERIMA","DISETUJUI"];
export class FindingError extends Error { kind:string; constructor(kind:string,message:string){super(message);this.kind=kind;} }
function fail():never{throw new FindingError("server","Data catatan tidak dapat dibaca. Coba muat ulang.");}
function idPath(id:number){if(!positive(id))throw new FindingError("validation","Kelompok tidak valid.");return id;}
export function validateFinding(d:FindingDraft):string|null {
 if(!["PPT","PDF","BUKU","PRESENTASI"].includes(d.fileType)||!["WAJIB","SARAN"].includes(d.priority))return "Pilih jenis tugas dan prioritas.";
 for(const [key,max] of [["location",200],["problem",2000],["action",2000]] as const){const text=d[key].trim();if(!text||[...text].length>max||/[\u0000-\u001f\u007f]/u.test(text))return "Isi lokasi, masalah, dan perbaikan dengan teks satu baris sesuai batas panjang.";}
 return null;
}
function parse(v:unknown):PublicFinding|null{
 if(!record(v)||!positive(v.id)||typeof v.resolved!=="boolean"||!date(v.updatedAt)||!date(v.publishedAt))return null;
 if(typeof v.location!=="string"||typeof v.problem!=="string"||typeof v.action!=="string"||typeof v.fileType!=="string"||typeof v.priority!=="string")return null;
 const d={fileType:v.fileType,location:v.location,problem:v.problem,action:v.action,priority:v.priority} as FindingDraft;
 if(validateFinding(d))return null;return {...d,id:v.id,resolved:v.resolved,publishedAt:v.publishedAt,updatedAt:v.updatedAt};
}
function adminParse(v:unknown):Finding|null{
 if(!record(v)||!positive(v.groupId)||!date(v.createdAt)||(v.publishedAt!==null&&!date(v.publishedAt)))return null;
 const f=parse({...v,publishedAt:v.publishedAt??v.createdAt});if(!f)return null;
 return {...f,groupId:v.groupId,publishedAt:v.publishedAt as string|null,createdAt:v.createdAt};
}
async function request(path:string,method:string,body:unknown,csrf:string|undefined,signal:AbortSignal|undefined,deps:Deps):Promise<unknown>{
 let r:Response;try{r=await(deps.fetch??fetch)(path,{method,credentials:"same-origin",cache:"no-store",signal,headers:csrf?{"Content-Type":"application/json","X-CSRF-Token":csrf}:undefined,body:body===undefined?undefined:JSON.stringify(body)});}catch(e){if(e instanceof Error&&e.name==="AbortError")throw e;throw new FindingError("network","Koneksi gagal. Coba lagi.");}
 if(r.status===204)return undefined;
 let v:unknown;try{v=await r.json();}catch{fail();}
 if(!r.ok){const code=record(v)&&record(v.error)?v.error.code:undefined;throw new FindingError(r.status===401?"unauthenticated":String(code??r.status),r.status===401?"Sesi berakhir. Silakan masuk kembali.":code==="REVIEW_RULE"?"Status belum sesuai: persetujuan perlu PPT dan PDF lengkap tanpa catatan wajib terbuka. Ubah ke Sedang Diperiksa sebelum mengoreksi catatan/berkas.":code==="FINDING_PUBLISHED"?"Isi catatan terbit tidak dapat diubah atau dihapus; hanya bisa ditandai selesai.":r.status===409?"Data berubah atau tindakan tidak sesuai. Muat ulang lalu coba lagi.":"Catatan gagal diproses. Periksa isian dan coba lagi.");}
 if(!record(v)||!("data"in v))fail();return v.data;
}
export async function getPublicFindings(id:number,signal?:AbortSignal,deps:Deps={}):Promise<PublicFinding[]>{
 const v=await request(`/api/v1/public/groups/${idPath(id)}`,"GET",undefined,undefined,signal,deps);if(!record(v)||!Array.isArray(v.findings))fail();return v.findings.map(x=>parse(x)??fail());
}
export async function listFindings(id:number,signal?:AbortSignal,deps:Deps={}):Promise<Finding[]>{const v=await request(`/api/v1/admin/groups/${idPath(id)}/findings`,"GET",undefined,undefined,signal,deps);if(!Array.isArray(v))fail();return v.map(x=>adminParse(x)??fail());}
export async function saveFinding(group:number,d:FindingDraft,csrf:string,existing:Finding|null=null,deps:Deps={}):Promise<Finding>{
 const error=validateFinding(d);if(error)throw new FindingError("validation",error);
 const v=await request(existing?`/api/v1/admin/findings/${idPath(existing.id)}`:`/api/v1/admin/groups/${idPath(group)}/findings`,existing?"PATCH":"POST",{fileType:d.fileType,location:d.location,problem:d.problem,action:d.action,priority:d.priority,...(existing?{resolved:existing.resolved,updatedAt:existing.updatedAt}:{})},csrf,undefined,deps);return adminParse(v)??fail();
}
export async function resolveFinding(f:Finding,csrf:string,deps:Deps={}):Promise<Finding>{return saveFinding(f.groupId,f,csrf,{...f,resolved:true},deps);}
export async function deleteFinding(f:Finding,csrf:string,deps:Deps={}):Promise<void>{await request(`/api/v1/admin/findings/${idPath(f.id)}?updatedAt=${encodeURIComponent(f.updatedAt)}`,"DELETE",undefined,csrf,undefined,deps);}
export async function getReview(id:number,signal?:AbortSignal,deps:Deps={}):Promise<ReviewRecord>{const v=await request(`/api/v1/admin/groups/${idPath(id)}/review-status`,"GET",undefined,undefined,signal,deps);if(!record(v)||!reviews.includes(String(v.status))||!date(v.updatedAt))fail();return {status:v.status as Review,updatedAt:v.updatedAt};}
export async function setReview(id:number,status:Review,updatedAt:string,csrf:string,deps:Deps={}):Promise<void>{await request(`/api/v1/admin/groups/${idPath(id)}/review-status`,"PATCH",{status,updatedAt},csrf,undefined,deps);}
export async function publishFindings(id:number,status:Review,updatedAt:string,csrf:string,deps:Deps={}):Promise<void>{await request(`/api/v1/admin/groups/${idPath(id)}/publish`,"POST",{status,updatedAt},csrf,undefined,deps);}
