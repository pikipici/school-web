"use client";

import { useCallback, useEffect, useRef, useState, type ChangeEvent, type FormEvent } from "react";

import {
  ACCEPT,
  ATTACHMENT_LIMITS,
  AttachmentError,
  KIND_LABEL,
  checkFile,
  deleteAttachment,
  formatSize,
  listAdminAttachments,
  messageForAttachment,
  uploadAttachment,
  type AdminAttachment
} from "@/lib/api-client/attachments";
import { formatWIB } from "@/lib/format";

type ListState = { status: "loading" } | { status: "error"; error: AttachmentError } | { status: "ready"; items: AdminAttachment[] };
type Upload = { status: "idle" } | { status: "sending"; name: string; sent: number; total: number };
type Notice = { tone: "error" | "success"; text: string } | null;

const asError = (e: unknown) => (e instanceof AttachmentError ? e : new AttachmentError("server"));
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";

/** Upload, list and delete attachments of one saved material. */
export function AttachmentManager({ materialId, csrf, onExpired }: { materialId: number; csrf: string; onExpired: () => void }) {
  const [list, setList] = useState<ListState>({ status: "loading" });
  const [tick, setTick] = useState(0);
  const [file, setFile] = useState<File | null>(null);
  const [upload, setUpload] = useState<Upload>({ status: "idle" });
  const [notice, setNotice] = useState<Notice>(null);
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const noticeRef = useRef<HTMLDivElement>(null);

  const fail = useCallback(
    (e: unknown) => {
      const error = asError(e);
      if (error.kind === "unauthenticated") onExpired();
      return error;
    },
    [onExpired]
  );

  useEffect(() => {
    const controller = new AbortController();
    listAdminAttachments(materialId, controller.signal).then(
      (items) => setList({ status: "ready", items }),
      (e: unknown) => {
        if (!isAbort(e)) setList({ status: "error", error: fail(e) });
      }
    );
    return () => controller.abort();
  }, [materialId, tick, fail]);

  useEffect(() => () => abortRef.current?.abort(), []);

  useEffect(() => {
    if (notice?.tone === "error") noticeRef.current?.focus();
  }, [notice]);

  const items = list.status === "ready" ? list.items : [];
  const full = items.length >= ATTACHMENT_LIMITS.perMaterial;
  const sending = upload.status === "sending";

  function onPick(event: ChangeEvent<HTMLInputElement>) {
    const picked = event.target.files?.[0] ?? null;
    setFile(picked);
    setNotice(null);
    if (picked) {
      const problem = checkFile(picked);
      if (problem) setNotice({ tone: "error", text: messageForAttachment(problem) });
    }
  }

  async function onUpload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!file || sending || full) return;
    const problem = checkFile(file);
    if (problem) {
      setNotice({ tone: "error", text: messageForAttachment(problem) });
      return;
    }
    const controller = new AbortController();
    abortRef.current = controller;
    setNotice(null);
    setUpload({ status: "sending", name: file.name, sent: 0, total: file.size });
    try {
      const item = await uploadAttachment(materialId, file, csrf, {
        signal: controller.signal,
        onProgress: (sent, total) => setUpload({ status: "sending", name: file.name, sent, total })
      });
      setList((l) => (l.status === "ready" ? { status: "ready", items: [...l.items, item] } : l));
      setNotice({ tone: "success", text: `“${item.name}” berhasil diunggah.` });
      setFile(null);
      if (inputRef.current) inputRef.current.value = "";
    } catch (e) {
      if (isAbort(e)) setNotice({ tone: "error", text: "Unggahan dibatalkan. Berkas tidak disimpan." });
      else {
        const error = fail(e);
        if (error.kind !== "unauthenticated") setNotice({ tone: "error", text: messageForAttachment(error.kind) });
        if (error.kind === "limit") setTick((n) => n + 1);
      }
    } finally {
      abortRef.current = null;
      setUpload({ status: "idle" });
    }
  }

  async function onDelete(item: AdminAttachment) {
    setDeleting(item.id);
    setNotice(null);
    try {
      await deleteAttachment(item.id, csrf);
      setList((l) => (l.status === "ready" ? { status: "ready", items: l.items.filter((x) => x.id !== item.id) } : l));
      setNotice({ tone: "success", text: `“${item.name}” dihapus.` });
    } catch (e) {
      const error = fail(e);
      if (error.kind === "not_found") setTick((n) => n + 1);
      else if (error.kind !== "unauthenticated") setNotice({ tone: "error", text: messageForAttachment(error.kind) });
    } finally {
      setDeleting(null);
      setConfirmId(null);
    }
  }

  const percent = upload.status === "sending" && upload.total > 0 ? Math.min(100, Math.round((upload.sent / upload.total) * 100)) : 0;

  return (
    <section className="content-section attachments-admin" aria-labelledby="lampiran-judul">
      <div className="section-heading">
        <h2 id="lampiran-judul">Lampiran</h2>
        {list.status === "ready" ? (
          <p className="field-hint">
            {items.length} dari {ATTACHMENT_LIMITS.perMaterial} lampiran
          </p>
        ) : null}
      </div>
      <p className="field-hint">PDF, PPT, atau PPTX, paling besar 25 MB per berkas. Presentasi bermakro (.pptm) ditolak.</p>

      <div ref={noticeRef} tabIndex={-1} aria-live="polite">
        {notice ? (
          <div className={notice.tone === "error" ? "form-error" : "form-success"} role={notice.tone === "error" ? "alert" : "status"}>
            <span className="notice-icon" aria-hidden="true">
              {notice.tone === "error" ? "!" : "✓"}
            </span>
            <p>{notice.text}</p>
          </div>
        ) : null}
      </div>

      {list.status === "loading" ? (
        <p className="public-state loading" role="status">
          Memuat lampiran…
        </p>
      ) : list.status === "error" ? (
        <div className="public-state error" role="alert">
          <p>{messageForAttachment(list.error.kind)}</p>
          <button className="retry-button" type="button" onClick={() => setTick((n) => n + 1)}>
            Coba lagi
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="empty-block" role="status">
          <p>Belum ada lampiran.</p>
        </div>
      ) : (
        <ul className="admin-list">
          {items.map((item) => (
            <li className="admin-row" key={item.id}>
              <div className="admin-row-main">
                <h3>{item.name}</h3>
                <p className="admin-row-meta">
                  <span>{KIND_LABEL[item.kind]}</span>
                  <span>{formatSize(item.size)}</span>
                  <span>Diunggah {formatWIB(item.createdAt)}</span>
                </p>
              </div>
              <div className="admin-row-actions">
                <button className="button-secondary" type="button" disabled={deleting !== null || sending} onClick={() => setConfirmId(item.id)}>
                  Hapus<span className="visually-hidden"> {item.name}</span>
                </button>
              </div>
              {confirmId === item.id ? (
                <div className="confirm-row" role="group" aria-label={`Konfirmasi hapus ${item.name}`}>
                  <p>Hapus “{item.name}”? Pengunjung tidak bisa mengunduhnya lagi.</p>
                  <button className="button-danger" type="button" disabled={deleting === item.id} onClick={() => onDelete(item)}>
                    {deleting === item.id ? "Menghapus…" : "Ya, hapus lampiran"}
                  </button>
                  <button className="button-secondary" type="button" disabled={deleting === item.id} onClick={() => setConfirmId(null)}>
                    Batal
                  </button>
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {list.status === "ready" ? (
        full ? (
          <p className="field-hint" role="status">
            Sudah 5 lampiran. Hapus satu untuk mengunggah berkas lain.
          </p>
        ) : (
          <form className="form-card form-wide upload-form" onSubmit={onUpload} aria-busy={sending}>
            <div className="form-field">
              <label htmlFor="berkas">Pilih berkas</label>
              <input id="berkas" ref={inputRef} type="file" accept={ACCEPT} disabled={sending} onChange={onPick} aria-describedby="berkas-petunjuk" />
              <p id="berkas-petunjuk" className="field-hint">
                {file ? `${file.name} · ${formatSize(file.size)}` : "Belum ada berkas dipilih."}
              </p>
            </div>
            {upload.status === "sending" ? (
              <div className="upload-progress" role="status">
                <label htmlFor="progres-unggah">
                  Mengunggah {upload.name}: {percent}% ({formatSize(upload.sent)} dari {formatSize(upload.total)})
                </label>
                <progress id="progres-unggah" max={100} value={percent}>
                  {percent}%
                </progress>
                <button className="button-secondary" type="button" onClick={() => abortRef.current?.abort()}>
                  Batalkan unggahan
                </button>
              </div>
            ) : (
              <button className="button-primary" type="submit" disabled={!file || checkFile(file) !== null}>
                Unggah lampiran
              </button>
            )}
          </form>
        )
      ) : null}
    </section>
  );
}
