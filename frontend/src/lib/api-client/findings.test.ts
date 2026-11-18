import { test } from "node:test";
import assert from "node:assert/strict";
import { getPublicFindings, listFindings, saveFinding, publishFindings, resolveFinding, validateFinding, type FindingDraft } from "./findings.ts";
const at="2026-11-18T08:00:00Z";
const draft:FindingDraft={fileType:"PPT",location:"Slide 5",problem:"Diagram salah",action:"Perbaiki simpul",priority:"WAJIB"};
const finding={...draft,id:1,groupId:2,resolved:false,publishedAt:at,createdAt:at,updatedAt:at};
test("public findings use anonymous endpoint and strip private keys",async()=>{
 const rows=await getPublicFindings(2,undefined,{fetch:async(url,init)=>{assert.equal(url,"/api/v1/public/groups/2");assert.equal(init?.cache,"no-store");return Response.json({data:{findings:[{...finding,note:"private"}]}})}});
 assert.deepEqual(Object.keys(rows[0]).sort(),["id","fileType","location","problem","action","priority","resolved","publishedAt","updatedAt"].sort());
});
test("public drafts and malformed findings are rejected",async()=>{
 for(const item of [{...finding,publishedAt:null},{...finding,priority:"OTHER"}])await assert.rejects(getPublicFindings(2,undefined,{fetch:async()=>Response.json({data:{findings:[item]}})}));
});
test("admin list permits drafts and save uses CSRF and conflict stamp",async()=>{
 const rows=await listFindings(2,undefined,{fetch:async()=>Response.json({data:[{...finding,publishedAt:null}]})});assert.equal(rows[0].publishedAt,null);
 await saveFinding(2,draft,"csrf",{...finding,publishedAt:null},{fetch:async(url,init)=>{assert.equal(url,"/api/v1/admin/findings/1");assert.equal(init?.method,"PATCH");assert.equal((init?.headers as Record<string,string>)["X-CSRF-Token"],"csrf");assert.equal(JSON.parse(init?.body as string).updatedAt,at);return Response.json({data:finding})}});
});
test("publication sends review and expected timestamp in one request",async()=>{
 await publishFindings(2,"PERLU_REVISI",at,"csrf",{fetch:async(url,init)=>{assert.equal(url,"/api/v1/admin/groups/2/publish");assert.deepEqual(JSON.parse(init?.body as string),{status:"PERLU_REVISI",updatedAt:at});return Response.json({data:{published:true}})}});
});
test("resolution sends no response-only keys",async()=>{await resolveFinding(finding,"csrf",{fetch:async(url,init)=>{const body=JSON.parse(init?.body as string);assert.deepEqual(Object.keys(body).sort(),["fileType","location","problem","action","priority","resolved","updatedAt"].sort());assert.equal(body.resolved,true);return Response.json({data:{...finding,resolved:true}})}});});
test("validation prevents empty corrections and oversize fields",()=>{assert.equal(validateFinding(draft),null);assert.ok(validateFinding({...draft,location:" "}));assert.ok(validateFinding({...draft,problem:"x".repeat(2001)}));});
