"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import { getTopics, type Topic } from "@/lib/api-client/content";
import {
  ProgressError,
  createGroup,
  deleteGroup,
  getClass,
  listGroups,
  listYears,
  messageForProgress,
  suggestEmail,
  updateGroup,
  validateGroup,
  type GroupDraft,
  type GroupErrors,
  type SchoolClass,
  type StudyGroup
} from "@/lib/api-client/progress";
import { validContentID } from "@/lib/format";

import { AdminPageGate } from "./AdminPageGate";
import { ActiveBadge } from "./ClassManager";

type Notice = { tone: "ok" | "error"; text: string } | null;

const asError = (e: unknown) => (e instanceof ProgressError ? e : new ProgressError("server"));
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";
const blank = (classId: number): GroupDraft => ({ classId, topicId: 0, name: "", email: "", sortOrder: 0, active: true });
const draftOf = (g: StudyGroup): GroupDraft => ({ classId: g.classId, topicId: g.topicId, name: g.name, email: g.email ?? "", sortOrder: g.sortOrder, active: g.active });

function GroupFields({
  draft,
  errors,
  set,
  topics,
  idPrefix,
  suggestion
}: {
  draft: GroupDraft;
  errors: GroupErrors;
  set: (d: GroupDraft) => void;
  topics: Topic[];
  idPrefix: string;
  suggestion: string;
}) {
  return (
    <>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-nama`}>Nama kelompok (wajib)</label>
        <input id={`${idPrefix}-nama`} type="text" maxLength={60} value={draft.name} aria-invalid={errors.name ? true : undefined} onChange={(e) => set({ ...draft, name: e.target.value })} />
        {errors.name ? <p className="field-hint">{errors.name}</p> : null}
      </div>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-topik`}>Topik (wajib)</label>
        <select id={`${idPrefix}-topik`} value={draft.topicId || ""} aria-invalid={errors.topicId ? true : undefined} onChange={(e) => set({ ...draft, topicId: Number(e.target.value) })}>
          <option value="">Pilih topik</option>
          {topics.map((t) => (
            <option key={t.id} value={t.id}>
              {t.code} — {t.title}
            </option>
          ))}
        </select>
        {errors.topicId ? <p className="field-hint">{errors.topicId}</p> : null}
      </div>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-email`}>Email kelompok (opsional, hanya terlihat guru)</label>
        <input
          id={`${idPrefix}-email`}
          type="email"
          maxLength={254}
          autoComplete="off"
          value={draft.email}
          aria-invalid={errors.email ? true : undefined}
          aria-describedby={`${idPrefix}-email-petunjuk`}
          onChange={(e) => set({ ...draft, email: e.target.value })}
        />
        <p id={`${idPrefix}-email-petunjuk`} className="field-hint">
          {errors.email ?? (suggestion ? `Format juknis: ${suggestion}` : "Format juknis: [nama].[kelas].2627.informatika@gmail.com")}
        </p>
        {suggestion && draft.email.trim() === "" ? (
          <button className="button-secondary" type="button" onClick={() => set({ ...draft, email: suggestion })}>
            Isi dengan format juknis
          </button>
        ) : null}
      </div>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-urutan`}>Urutan tampil</label>
        <input
          id={`${idPrefix}-urutan`}
          type="number"
          inputMode="numeric"
          min={0}
          step={1}
          value={Number.isNaN(draft.sortOrder) ? "" : String(draft.sortOrder)}
          aria-invalid={errors.sortOrder ? true : undefined}
          onChange={(e) => set({ ...draft, sortOrder: e.target.value === "" ? Number.NaN : Number(e.target.value) })}
        />
        {errors.sortOrder ? <p className="field-hint">{errors.sortOrder}</p> : null}
      </div>
    </>
  );
}

function Manager({ classId, csrf, onExpired }: { classId: number; csrf: string; onExpired: () => void }) {
  const [cls, setCls] = useState<SchoolClass | null>(null);
  const [yearLabel, setYearLabel] = useState("");
  const [topics, setTopics] = useState<Topic[]>([]);
  const [groups, setGroups] = useState<StudyGroup[] | null>(null);
  const [loadError, setLoadError] = useState<ProgressError | null>(null);
  const [tick, setTick] = useState(0);
  const [notice, setNotice] = useState<Notice>(null);
  const [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState<GroupDraft>(blank(classId));
  const [errors, setErrors] = useState<GroupErrors>({});
  const [editId, setEditId] = useState<number | null>(null);
  const [edit, setEdit] = useState<GroupDraft>(blank(classId));
  const [editErrors, setEditErrors] = useState<GroupErrors>({});
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const noticeRef = useRef<HTMLDivElement>(null);

  const reload = () => setTick((n) => n + 1);

  useEffect(() => {
    const c = new AbortController();
    const onFail = (e: unknown) => {
      if (isAbort(e)) return;
      const error = asError(e);
      if (error.kind === "unauthenticated") onExpired();
      else setLoadError(error);
    };
    getClass(classId, c.signal).then((k) => {
      setCls(k);
      listYears(c.signal).then((ys) => setYearLabel(ys.find((y) => y.id === k.yearId)?.label ?? ""), () => undefined);
    }, onFail);
    listGroups(classId, c.signal).then(setGroups, onFail);
    getTopics(c.signal).then(setTopics, () => undefined);
    return () => c.abort();
  }, [classId, tick, onExpired]);

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
        if (error.kind === "conflict" || error.kind === "not_found") reload();
      }
    } finally {
      setBusy(false);
    }
  }

  function onAdd(e: FormEvent) {
    e.preventDefault();
    const found = validateGroup(draft);
    setErrors(found);
    if (Object.keys(found).length) return;
    run(async () => {
      const g = await createGroup(draft, csrf);
      setDraft(blank(classId));
      return `${g.name} ditambahkan dan tampil di publik sebagai Belum Mengumpulkan.`;
    });
  }

  function onSave(e: FormEvent, g: StudyGroup) {
    e.preventDefault();
    const found = validateGroup(edit);
    setEditErrors(found);
    if (Object.keys(found).length) return;
    run(async () => {
      const saved = await updateGroup(g.id, edit, g.updatedAt, csrf);
      setEditId(null);
      return `${saved.name} disimpan.`;
    });
  }

  const toggle = (g: StudyGroup) =>
    run(async () => {
      const saved = await updateGroup(g.id, { ...draftOf(g), active: !g.active }, g.updatedAt, csrf);
      return saved.active ? `${saved.name} aktif lagi.` : `${saved.name} dinonaktifkan dan disembunyikan dari publik.`;
    });

  const remove = (g: StudyGroup) =>
    run(async () => {
      setConfirmId(null);
      await deleteGroup(g.id, csrf);
      return `${g.name} dihapus.`;
    });

  if (loadError) {
    return (
      <div className="public-state error" role="alert">
        <p>{messageForProgress(loadError.kind)}</p>
        {loadError.kind !== "not_found" ? (
          <button className="retry-button" type="button" onClick={() => { setLoadError(null); reload(); }}>
            Coba lagi
          </button>
        ) : null}
        <p>
          <Link href="/admin/kelas/">Kembali ke daftar kelas</Link>
        </p>
      </div>
    );
  }
  if (!cls || !groups) {
    return (
      <p className="public-state loading" role="status">
        Memuat kelompok…
      </p>
    );
  }

  const topicName = (id: number) => {
    const t = topics.find((x) => x.id === id);
    return t ? `${t.code} — ${t.title}` : "Topik";
  };

  return (
    <>
      <h1 className="page-title">
        Kelompok kelas {cls.code}
        {yearLabel ? ` (${yearLabel})` : ""}
      </h1>
      <p>
        <Link href="/admin/kelas/">Kembali ke daftar kelas</Link>
      </p>
      {!cls.active ? <p className="section-note">Kelas ini nonaktif, jadi kelompoknya tidak tampil di tabel publik.</p> : null}

      <div ref={noticeRef} tabIndex={-1} aria-live="polite">
        {notice ? (
          <p className={notice.tone === "ok" ? "form-success" : "form-error"} role={notice.tone === "error" ? "alert" : "status"}>
            <span aria-hidden="true">{notice.tone === "ok" ? "✓ " : "! "}</span>
            {notice.text}
          </p>
        ) : null}
      </div>

      <section className="content-section" aria-labelledby="daftar-kelompok">
        <h2 id="daftar-kelompok">Daftar kelompok</h2>
        {groups.length === 0 ? (
          <div className="empty-block" role="status">
            <p>Belum ada kelompok di kelas ini. Tambahkan kelompok pertama di bawah.</p>
          </div>
        ) : (
          <ul className="admin-list">
            {groups.map((g) => (
              <li className="admin-row" key={g.id}>
                <div className="admin-row-main">
                  <h3>{g.name}</h3>
                  <p className="admin-row-meta">
                    <ActiveBadge active={g.active} />
                    <span>{topicName(g.topicId)}</span>
                    <span>{g.email ?? "Tanpa email"}</span>
                    <span>Urutan {g.sortOrder}</span>
                  </p>
                </div>
                {editId === g.id ? (
                  <form className="inline-form" onSubmit={(e) => onSave(e, g)} noValidate aria-label={`Ubah ${g.name}`}>
                    <GroupFields draft={edit} errors={editErrors} set={setEdit} topics={topics} idPrefix={`ubah-${g.id}`} suggestion={suggestEmail(edit.name, cls.code, yearLabel)} />
                    <div className="form-actions">
                      <button className="button-primary" type="submit" disabled={busy}>
                        {busy ? "Menyimpan…" : "Simpan kelompok"}
                      </button>
                      <button className="button-secondary" type="button" onClick={() => setEditId(null)}>
                        Batal
                      </button>
                    </div>
                  </form>
                ) : confirmId === g.id ? (
                  <div className="confirm-row" role="group" aria-label={`Konfirmasi hapus ${g.name}`}>
                    <p>Hapus {g.name}? Tindakan ini tidak dapat dibatalkan.</p>
                    <button className="button-danger" type="button" disabled={busy} onClick={() => remove(g)}>
                      Ya, hapus kelompok
                    </button>
                    <button className="button-secondary" type="button" onClick={() => setConfirmId(null)}>
                      Batal
                    </button>
                  </div>
                ) : (
                  <div className="admin-row-actions">
                    <button className="button-secondary" type="button" disabled={busy} onClick={() => { setEditId(g.id); setEdit(draftOf(g)); setEditErrors({}); }}>
                      Ubah<span className="visually-hidden"> {g.name}</span>
                    </button>
                    <button className="button-secondary" type="button" disabled={busy} onClick={() => toggle(g)}>
                      {g.active ? "Nonaktifkan" : "Aktifkan"}
                      <span className="visually-hidden"> {g.name}</span>
                    </button>
                    <button className="button-secondary" type="button" disabled={busy} onClick={() => setConfirmId(g.id)}>
                      Hapus<span className="visually-hidden"> {g.name}</span>
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}

        <form className="form-card form-wide inline-form" onSubmit={onAdd} noValidate aria-labelledby="tambah-kelompok">
          <h3 id="tambah-kelompok" className="form-actions">
            Tambah kelompok
          </h3>
          <GroupFields draft={draft} errors={errors} set={setDraft} topics={topics} idPrefix="baru" suggestion={suggestEmail(draft.name, cls.code, yearLabel)} />
          <div className="form-actions">
            <button className="button-primary" type="submit" disabled={busy}>
              Tambah kelompok
            </button>
          </div>
        </form>
      </section>
    </>
  );
}

export function GroupManagerPage() {
  const raw = useSearchParams().get("kelas");
  const classId = validContentID(raw);
  return (
    <AdminPageGate>
      {(csrf, onExpired) =>
        classId === null ? (
          <div className="public-state error" role="alert">
            <p>Kelas tidak dipilih atau ID-nya tidak valid.</p>
            <Link href="/admin/kelas/">Pilih kelas</Link>
          </div>
        ) : (
          <Manager classId={classId} csrf={csrf} onExpired={onExpired} />
        )
      }
    </AdminPageGate>
  );
}
