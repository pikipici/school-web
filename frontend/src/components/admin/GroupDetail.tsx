"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import { getTopics, type Topic } from "@/lib/api-client/content";
import {
  NOTEBOOK_LABEL,
  ProgressError,
  ROUND_LIMITS,
  SUBMISSION_LABEL,
  deleteRound,
  draftFromRound,
  getClass,
  getGroup,
  getNotebook,
  listRounds,
  messageForProgress,
  putNotebook,
  recordRound,
  roundLabel,
  suggestSubject,
  toWIBInput,
  updateRound,
  validateRound,
  type GroupNotebook,
  type RoundDraft,
  type RoundErrors,
  type SchoolClass,
  type StudyGroup,
  type Submission,
  type SubmissionRound
} from "@/lib/api-client/progress";
import { formatWIB, validContentID } from "@/lib/format";

import { AdminPageGate } from "./AdminPageGate";

type Notice = { tone: "ok" | "error"; text: string } | null;
type Loaded = { group: StudyGroup; cls: SchoolClass; topic: Topic | null; submission: Submission; rounds: SubmissionRound[]; notebook: GroupNotebook };

const asError = (e: unknown) => (e instanceof ProgressError ? e : new ProgressError("server"));
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";
const SUBMISSION_TONE: Record<Submission, string> = { BELUM: "lane-none", SEBAGIAN: "lane-partial", LENGKAP: "lane-done" };
const SUBMISSION_ICON: Record<Submission, string> = { BELUM: "○", SEBAGIAN: "◐", LENGKAP: "●" };

function blankDraft(g: StudyGroup): RoundDraft {
  return { receivedAt: toWIBInput(new Date()), senderEmail: g.email ?? "", subject: "", ppt: false, pdf: false, pptFilename: "", pdfFilename: "", note: "" };
}

function RoundFields({ draft, errors, set, idPrefix }: { draft: RoundDraft; errors: RoundErrors; set: (d: RoundDraft) => void; idPrefix: string }) {
  const hint = (id: string, text?: string) => (text ? <p id={id} className="field-hint field-error">{text}</p> : null);
  return (
    <>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-waktu`}>Email diterima (WIB, wajib)</label>
        <input
          id={`${idPrefix}-waktu`}
          type="datetime-local"
          value={draft.receivedAt}
          aria-invalid={errors.receivedAt ? true : undefined}
          aria-describedby={errors.receivedAt ? `${idPrefix}-waktu-galat` : undefined}
          onChange={(e) => set({ ...draft, receivedAt: e.target.value })}
        />
        {hint(`${idPrefix}-waktu-galat`, errors.receivedAt)}
      </div>
      <fieldset className="form-field" aria-describedby={errors.files ? `${idPrefix}-berkas-galat` : undefined}>
        <legend>Berkas yang diterima (minimal satu)</legend>
        <div className="form-check">
          <input id={`${idPrefix}-ppt`} type="checkbox" checked={draft.ppt} onChange={(e) => set({ ...draft, ppt: e.target.checked, pptFilename: e.target.checked ? draft.pptFilename : "" })} />
          <label htmlFor={`${idPrefix}-ppt`}>PPT</label>
        </div>
        <div className="form-check">
          <input id={`${idPrefix}-pdf`} type="checkbox" checked={draft.pdf} onChange={(e) => set({ ...draft, pdf: e.target.checked, pdfFilename: e.target.checked ? draft.pdfFilename : "" })} />
          <label htmlFor={`${idPrefix}-pdf`}>PDF</label>
        </div>
        {hint(`${idPrefix}-berkas-galat`, errors.files)}
      </fieldset>
      {draft.ppt ? (
        <div className="form-field">
          <label htmlFor={`${idPrefix}-nama-ppt`}>Nama berkas PPT (opsional)</label>
          <input id={`${idPrefix}-nama-ppt`} type="text" maxLength={ROUND_LIMITS.fileName} value={draft.pptFilename} aria-invalid={errors.pptFilename ? true : undefined} onChange={(e) => set({ ...draft, pptFilename: e.target.value })} />
          {hint(`${idPrefix}-nama-ppt-galat`, errors.pptFilename)}
        </div>
      ) : null}
      {draft.pdf ? (
        <div className="form-field">
          <label htmlFor={`${idPrefix}-nama-pdf`}>Nama berkas PDF (opsional)</label>
          <input id={`${idPrefix}-nama-pdf`} type="text" maxLength={ROUND_LIMITS.fileName} value={draft.pdfFilename} aria-invalid={errors.pdfFilename ? true : undefined} onChange={(e) => set({ ...draft, pdfFilename: e.target.value })} />
          {hint(`${idPrefix}-nama-pdf-galat`, errors.pdfFilename)}
        </div>
      ) : null}
      <div className="form-field">
        <label htmlFor={`${idPrefix}-pengirim`}>Alamat pengirim (opsional)</label>
        <input id={`${idPrefix}-pengirim`} type="email" autoComplete="off" maxLength={254} value={draft.senderEmail} aria-invalid={errors.senderEmail ? true : undefined} onChange={(e) => set({ ...draft, senderEmail: e.target.value })} />
        {hint(`${idPrefix}-pengirim-galat`, errors.senderEmail)}
      </div>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-subjek`}>Subjek email (opsional)</label>
        <input id={`${idPrefix}-subjek`} type="text" maxLength={ROUND_LIMITS.subject} value={draft.subject} aria-invalid={errors.subject ? true : undefined} onChange={(e) => set({ ...draft, subject: e.target.value })} />
        {hint(`${idPrefix}-subjek-galat`, errors.subject)}
      </div>
      <div className="form-field form-span">
        <label htmlFor={`${idPrefix}-catatan`}>Catatan guru (opsional, tidak terlihat siswa)</label>
        <textarea id={`${idPrefix}-catatan`} rows={3} maxLength={ROUND_LIMITS.note} value={draft.note} aria-invalid={errors.note ? true : undefined} onChange={(e) => set({ ...draft, note: e.target.value })} />
        {hint(`${idPrefix}-catatan-galat`, errors.note)}
      </div>
    </>
  );
}

function Detail({ groupId, csrf, onExpired }: { groupId: number; csrf: string; onExpired: () => void }) {
  const [state, setState] = useState<Loaded | null>(null);
  const [loadError, setLoadError] = useState<ProgressError | null>(null);
  const [tick, setTick] = useState(0);
  const [notice, setNotice] = useState<Notice>(null);
  const [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState<RoundDraft | null>(null);
  const [errors, setErrors] = useState<RoundErrors>({});
  const [editId, setEditId] = useState<number | null>(null);
  const [edit, setEdit] = useState<RoundDraft | null>(null);
  const [editErrors, setEditErrors] = useState<RoundErrors>({});
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [book, setBook] = useState<{ received: boolean; receivedAt: string; note: string } | null>(null);
  const noticeRef = useRef<HTMLDivElement>(null);

  const reload = () => setTick((n) => n + 1);

  useEffect(() => {
    const c = new AbortController();
    (async () => {
      try {
        const group = await getGroup(groupId, c.signal);
        const [cls, rounds, notebook, topics] = await Promise.all([
          getClass(group.classId, c.signal),
          listRounds(groupId, c.signal),
          getNotebook(groupId, c.signal),
          getTopics(c.signal).catch(() => [] as Topic[])
        ]);
        setState({ group, cls, topic: topics.find((t) => t.id === group.topicId) ?? null, submission: rounds.submission, rounds: rounds.rounds, notebook });
        setDraft((d) => d ?? blankDraft(group));
        setBook({ received: notebook.received, receivedAt: notebook.receivedAt ? toWIBInput(notebook.receivedAt) : "", note: notebook.note });
        setLoadError(null);
      } catch (e) {
        if (isAbort(e)) return;
        const error = asError(e);
        if (error.kind === "unauthenticated") onExpired();
        else setLoadError(error);
      }
    })();
    return () => c.abort();
  }, [groupId, tick, onExpired]);

  useEffect(() => {
    if (notice?.tone === "error") noticeRef.current?.focus();
  }, [notice]);

  async function run(action: () => Promise<string>) {
    setBusy(true);
    setNotice(null);
    try {
      setNotice({ tone: "ok", text: await action() });
      reload();
    } catch (e) {
      const error = asError(e);
      if (error.kind === "unauthenticated") onExpired();
      else {
        setNotice({ tone: "error", text: messageForProgress(error.kind) });
        if (error.kind === "conflict" || error.kind === "not_found" || error.kind === "not_last") reload();
      }
    } finally {
      setBusy(false);
    }
  }

  if (loadError) {
    return (
      <div className="public-state error" role="alert">
        <p>{messageForProgress(loadError.kind)}</p>
        {loadError.kind !== "not_found" ? (
          <button className="retry-button" type="button" onClick={reload}>
            Coba lagi
          </button>
        ) : null}
        <p>
          <Link href="/admin/kelas/">Kembali ke daftar kelas</Link>
        </p>
      </div>
    );
  }
  if (!state || !draft || !book) {
    return (
      <p className="public-state loading" role="status">
        Memuat kelompok…
      </p>
    );
  }

  const { group, cls, topic, submission, rounds, notebook } = state;
  const next = rounds.length === 0 ? 0 : rounds[rounds.length - 1].roundNumber + 1;

  function onRecord(e: FormEvent) {
    e.preventDefault();
    if (!draft) return;
    const found = validateRound(draft);
    setErrors(found);
    if (Object.keys(found).length) return;
    run(async () => {
      const r = await recordRound(groupId, next, draft, csrf);
      setDraft(blankDraft(group));
      setErrors({});
      return `${roundLabel(r.roundNumber)} dicatat.`;
    });
  }

  function onSaveEdit(e: FormEvent, r: SubmissionRound) {
    e.preventDefault();
    if (!edit) return;
    const found = validateRound(edit);
    setEditErrors(found);
    if (Object.keys(found).length) return;
    run(async () => {
      await updateRound(r.id, edit, r.updatedAt, csrf);
      setEditId(null);
      return `${roundLabel(r.roundNumber)} disimpan.`;
    });
  }

  function onSaveBook(e: FormEvent) {
    e.preventDefault();
    if (!book) return;
    run(async () => {
      const n = await putNotebook(groupId, book, csrf);
      return n.received ? "Buku tulis dicatat sudah diterima." : "Buku tulis dicatat belum diterima.";
    });
  }

  return (
    <>
      <h1 className="page-title">{group.name}</h1>
      <p>
        <Link href={`/admin/kelompok/?kelas=${cls.id}`}>Kembali ke kelompok kelas {cls.code}</Link>
      </p>
      <p className="admin-row-meta">
        <span>Kelas {cls.code}</span>
        <span>{topic ? `${topic.code} — ${topic.title}` : "Topik"}</span>
        <span>{group.email ?? "Tanpa email kelompok"}</span>
      </p>

      <div ref={noticeRef} tabIndex={-1} aria-live="polite">
        {notice ? (
          <p className={notice.tone === "ok" ? "form-success" : "form-error"} role={notice.tone === "error" ? "alert" : "status"}>
            <span aria-hidden="true">{notice.tone === "ok" ? "✓ " : "! "}</span>
            {notice.text}
          </p>
        ) : null}
      </div>

      <section className="content-section" aria-labelledby="jalur-pengumpulan">
        <div className="section-heading">
          <h2 id="jalur-pengumpulan">Pengumpulan PPT dan PDF</h2>
          <span className={`lane-chip ${SUBMISSION_TONE[submission]}`}>
            <span aria-hidden="true">{SUBMISSION_ICON[submission]}</span> {SUBMISSION_LABEL[submission]}
          </span>
        </div>
        <p className="field-hint">Status dihitung dari semua putaran: revisi yang hanya berisi PPT tidak menghapus PDF yang sudah diterima.</p>

        {rounds.length === 0 ? (
          <div className="empty-block" role="status">
            <p>Belum ada pengumpulan yang dicatat.</p>
          </div>
        ) : (
          <ol className="admin-list round-list" reversed>
            {[...rounds].reverse().map((r) => {
              const isLast = r.roundNumber === next - 1;
              return (
                <li className="admin-row" key={r.id}>
                  <div className="admin-row-main">
                    <h3>{roundLabel(r.roundNumber)}</h3>
                    <p className="admin-row-meta">
                      <span>Diterima {formatWIB(r.receivedAt)}</span>
                      {r.ppt ? <span className="lane-chip lane-done"><span aria-hidden="true">✓</span> PPT{r.pptFilename ? `: ${r.pptFilename}` : ""}</span> : null}
                      {r.pdf ? <span className="lane-chip lane-done"><span aria-hidden="true">✓</span> PDF{r.pdfFilename ? `: ${r.pdfFilename}` : ""}</span> : null}
                    </p>
                    {r.senderEmail || r.subject ? (
                      <p className="admin-row-meta">
                        {r.senderEmail ? <span>Dari {r.senderEmail}</span> : null}
                        {r.subject ? <span>“{r.subject}”</span> : null}
                      </p>
                    ) : null}
                    {r.note ? <p className="field-hint">Catatan: {r.note}</p> : null}
                  </div>
                  {editId === r.id && edit ? (
                    <form className="inline-form" onSubmit={(e) => onSaveEdit(e, r)} noValidate aria-label={`Ubah ${roundLabel(r.roundNumber)}`}>
                      <RoundFields draft={edit} errors={editErrors} set={setEdit} idPrefix={`ubah-${r.id}`} />
                      <div className="form-actions">
                        <button className="button-primary" type="submit" disabled={busy}>
                          {busy ? "Menyimpan…" : "Simpan putaran"}
                        </button>
                        <button className="button-secondary" type="button" onClick={() => setEditId(null)}>
                          Batal
                        </button>
                      </div>
                    </form>
                  ) : confirmId === r.id ? (
                    <div className="confirm-row" role="group" aria-label={`Konfirmasi hapus ${roundLabel(r.roundNumber)}`}>
                      <p>Hapus {roundLabel(r.roundNumber)}? Gunakan hanya untuk membatalkan salah input.</p>
                      <button className="button-danger" type="button" disabled={busy} onClick={() => { setConfirmId(null); run(async () => { await deleteRound(r.id, csrf); return `${roundLabel(r.roundNumber)} dihapus.`; }); }}>
                        Ya, hapus putaran
                      </button>
                      <button className="button-secondary" type="button" onClick={() => setConfirmId(null)}>
                        Batal
                      </button>
                    </div>
                  ) : (
                    <div className="admin-row-actions">
                      <button className="button-secondary" type="button" disabled={busy} onClick={() => { setEditId(r.id); setEdit(draftFromRound(r)); setEditErrors({}); }}>
                        Ubah<span className="visually-hidden"> {roundLabel(r.roundNumber)}</span>
                      </button>
                      {isLast ? (
                        <button className="button-secondary" type="button" disabled={busy} onClick={() => setConfirmId(r.id)}>
                          Hapus<span className="visually-hidden"> {roundLabel(r.roundNumber)}</span>
                        </button>
                      ) : null}
                    </div>
                  )}
                </li>
              );
            })}
          </ol>
        )}

        <form className="form-card form-wide inline-form" onSubmit={onRecord} noValidate aria-labelledby="catat-pengumpulan">
          <h3 id="catat-pengumpulan" className="form-actions">
            Catat Pengumpulan: {roundLabel(next)}
          </h3>
          <p className="field-hint form-actions">Isi setelah Anda memeriksa Gmail sendiri. Website tidak membaca email.</p>
          <RoundFields draft={draft} errors={errors} set={setDraft} idPrefix="baru" />
          {draft.subject === "" ? (
            <div className="form-actions">
              <button className="button-secondary" type="button" onClick={() => setDraft({ ...draft, subject: suggestSubject(next, cls.code, group.name, topic?.title ?? "") })}>
                Isi subjek format juknis
              </button>
            </div>
          ) : null}
          <div className="form-actions">
            <button className="button-primary" type="submit" disabled={busy}>
              {busy ? "Menyimpan…" : `Catat ${roundLabel(next).toLowerCase()}`}
            </button>
          </div>
        </form>
      </section>

      <section className="content-section" aria-labelledby="jalur-buku">
        <div className="section-heading">
          <h2 id="jalur-buku">Buku tulis</h2>
          <span className={`lane-chip ${notebook.received ? "lane-done" : "lane-none"}`}>
            <span aria-hidden="true">{notebook.received ? "✓" : "○"}</span> {NOTEBOOK_LABEL[notebook.received ? "SUDAH_DITERIMA" : "BELUM_DITERIMA"]}
          </span>
        </div>
        <p className="field-hint">Satu buku catatan per kelompok, diserahkan langsung ke guru.</p>
        <form className="form-card form-wide inline-form" onSubmit={onSaveBook} noValidate aria-label="Catat buku tulis">
          <div className="form-check form-actions">
            <input id="buku-diterima" type="checkbox" checked={book.received} onChange={(e) => setBook({ ...book, received: e.target.checked, receivedAt: e.target.checked ? book.receivedAt || toWIBInput(new Date()) : "" })} />
            <label htmlFor="buku-diterima">Buku tulis sudah diterima</label>
          </div>
          {book.received ? (
            <div className="form-field">
              <label htmlFor="buku-waktu">Tanggal diterima (WIB)</label>
              <input id="buku-waktu" type="datetime-local" value={book.receivedAt} onChange={(e) => setBook({ ...book, receivedAt: e.target.value })} />
            </div>
          ) : null}
          <div className="form-field form-span">
            <label htmlFor="buku-catatan">Catatan (opsional)</label>
            <textarea id="buku-catatan" rows={2} maxLength={ROUND_LIMITS.notebookNote} value={book.note} onChange={(e) => setBook({ ...book, note: e.target.value })} />
          </div>
          <div className="form-actions">
            <button className="button-primary" type="submit" disabled={busy}>
              {busy ? "Menyimpan…" : "Simpan buku tulis"}
            </button>
          </div>
        </form>
        {notebook.updatedAt ? <p className="field-hint">Terakhir dicatat {formatWIB(notebook.updatedAt)}.</p> : null}
      </section>
    </>
  );
}

export function GroupDetailPage() {
  const id = validContentID(useSearchParams().get("id"));
  return (
    <AdminPageGate>
      {(csrf, onExpired) =>
        id === null ? (
          <div className="public-state error" role="alert">
            <p>ID kelompok tidak ada atau tidak valid.</p>
            <Link href="/admin/kelas/">Pilih kelas</Link>
          </div>
        ) : (
          <Detail groupId={id} csrf={csrf} onExpired={onExpired} />
        )
      }
    </AdminPageGate>
  );
}
