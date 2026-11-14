"use client";

import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";

import {
  ProgressError,
  activateYear,
  createClass,
  createYear,
  deleteClass,
  listClasses,
  listYears,
  messageForProgress,
  updateClass,
  validateClass,
  validateYearLabel,
  type AcademicYear,
  type ClassDraft,
  type ClassErrors,
  type SchoolClass
} from "@/lib/api-client/progress";

import { AdminPageGate } from "./AdminPageGate";

type Notice = { tone: "ok" | "error"; text: string } | null;

const asError = (e: unknown) => (e instanceof ProgressError ? e : new ProgressError("server"));
const isAbort = (e: unknown) => e instanceof DOMException && e.name === "AbortError";

export function ActiveBadge({ active }: { active: boolean }) {
  return active ? (
    <span className="status-badge status-published">
      <span aria-hidden="true">✓</span> Aktif
    </span>
  ) : (
    <span className="status-badge status-inactive">
      <span aria-hidden="true">–</span> Nonaktif
    </span>
  );
}

function ClassFields({ draft, errors, set, idPrefix }: { draft: ClassDraft; errors: ClassErrors; set: (d: ClassDraft) => void; idPrefix: string }) {
  return (
    <>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-kode`}>Kode kelas (wajib)</label>
        <input
          id={`${idPrefix}-kode`}
          type="text"
          maxLength={20}
          value={draft.code}
          aria-invalid={errors.code ? true : undefined}
          aria-describedby={`${idPrefix}-kode-petunjuk`}
          onChange={(e) => set({ ...draft, code: e.target.value })}
        />
        <p id={`${idPrefix}-kode-petunjuk`} className="field-hint">
          {errors.code ?? "Contoh: 9A. Tampil di tabel publik."}
        </p>
      </div>
      <div className="form-field">
        <label htmlFor={`${idPrefix}-nama`}>Nama kelas (opsional)</label>
        <input id={`${idPrefix}-nama`} type="text" maxLength={80} value={draft.name} aria-invalid={errors.name ? true : undefined} onChange={(e) => set({ ...draft, name: e.target.value })} />
        {errors.name ? <p className="field-hint">{errors.name}</p> : null}
      </div>
    </>
  );
}

function Manager({ csrf, onExpired }: { csrf: string; onExpired: () => void }) {
  const [years, setYears] = useState<AcademicYear[] | null>(null);
  const [yearId, setYearId] = useState<number | null>(null);
  const [classes, setClasses] = useState<SchoolClass[] | null>(null);
  const [loadError, setLoadError] = useState<ProgressError | null>(null);
  const [tick, setTick] = useState(0);
  const [notice, setNotice] = useState<Notice>(null);
  const [busy, setBusy] = useState(false);
  const [newYear, setNewYear] = useState("");
  const [newClass, setNewClass] = useState<ClassDraft>({ yearId: 0, code: "", name: "", active: true });
  const [newErrors, setNewErrors] = useState<ClassErrors>({});
  const [editId, setEditId] = useState<number | null>(null);
  const [edit, setEdit] = useState<ClassDraft>({ yearId: 0, code: "", name: "", active: true });
  const [editErrors, setEditErrors] = useState<ClassErrors>({});
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [confirmYear, setConfirmYear] = useState(false);
  const noticeRef = useRef<HTMLDivElement>(null);

  const reload = () => setTick((n) => n + 1);

  function fail(e: unknown) {
    const error = asError(e);
    if (error.kind === "unauthenticated") return onExpired();
    setNotice({ tone: "error", text: messageForProgress(error.kind) });
    if (error.kind === "conflict" || error.kind === "not_found") reload();
  }

  useEffect(() => {
    const c = new AbortController();
    listYears(c.signal).then(
      (ys) => {
        setYears(ys);
        setYearId((cur) => (cur !== null && ys.some((y) => y.id === cur) ? cur : (ys.find((y) => y.active)?.id ?? ys[0]?.id ?? null)));
      },
      (e: unknown) => {
        if (isAbort(e)) return;
        const error = asError(e);
        if (error.kind === "unauthenticated") onExpired();
        else setLoadError(error);
      }
    );
    return () => c.abort();
  }, [tick, onExpired]);

  useEffect(() => {
    if (yearId === null) return;
    const c = new AbortController();
    listClasses(yearId, c.signal).then(setClasses, (e: unknown) => {
      if (isAbort(e)) return;
      const error = asError(e);
      if (error.kind === "unauthenticated") onExpired();
      else setLoadError(error);
    });
    return () => c.abort();
  }, [yearId, tick, onExpired]);

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
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  function onAddYear(e: FormEvent) {
    e.preventDefault();
    const problem = validateYearLabel(newYear);
    if (problem) return setNotice({ tone: "error", text: problem });
    run(async () => {
      const y = await createYear(newYear, csrf);
      setNewYear("");
      setYearId(y.id);
      return `Tahun ajaran ${y.label} ditambahkan (belum aktif).`;
    });
  }

  function onAddClass(e: FormEvent) {
    e.preventDefault();
    if (yearId === null) return;
    const d = { ...newClass, yearId };
    const errors = validateClass(d);
    setNewErrors(errors);
    if (Object.keys(errors).length) return;
    run(async () => {
      const c = await createClass(d, csrf);
      setNewClass({ yearId: 0, code: "", name: "", active: true });
      return `Kelas ${c.code} ditambahkan.`;
    });
  }

  function onSaveEdit(e: FormEvent, c: SchoolClass) {
    e.preventDefault();
    const errors = validateClass(edit);
    setEditErrors(errors);
    if (Object.keys(errors).length) return;
    run(async () => {
      const saved = await updateClass(c.id, edit, c.updatedAt, csrf);
      setEditId(null);
      return `Kelas ${saved.code} disimpan.`;
    });
  }

  const toggle = (c: SchoolClass) =>
    run(async () => {
      const saved = await updateClass(c.id, { yearId: c.yearId, code: c.code, name: c.name, active: !c.active }, c.updatedAt, csrf);
      return saved.active ? `Kelas ${saved.code} aktif lagi dan tampil di publik.` : `Kelas ${saved.code} dinonaktifkan dan disembunyikan dari publik.`;
    });

  const remove = (c: SchoolClass) =>
    run(async () => {
      setConfirmId(null);
      await deleteClass(c.id, csrf);
      return `Kelas ${c.code} dihapus.`;
    });

  if (loadError) {
    return (
      <div className="public-state error" role="alert">
        <p>{messageForProgress(loadError.kind)}</p>
        <button className="retry-button" type="button" onClick={() => { setLoadError(null); reload(); }}>
          Coba lagi
        </button>
      </div>
    );
  }
  if (!years) {
    return (
      <p className="public-state loading" role="status">
        Memuat tahun ajaran…
      </p>
    );
  }

  const year = years.find((y) => y.id === yearId) ?? null;

  return (
    <>
      <div ref={noticeRef} tabIndex={-1} aria-live="polite">
        {notice ? (
          <p className={notice.tone === "ok" ? "form-success" : "form-error"} role={notice.tone === "error" ? "alert" : "status"}>
            <span aria-hidden="true">{notice.tone === "ok" ? "✓ " : "! "}</span>
            {notice.text}
          </p>
        ) : null}
      </div>

      <section className="content-section" aria-labelledby="tahun-ajaran">
        <h2 id="tahun-ajaran">Tahun ajaran</h2>
        <div className="inline-form">
          <div className="form-field">
            <label htmlFor="pilih-tahun">Tampilkan kelas tahun ajaran</label>
            <select id="pilih-tahun" value={yearId ?? ""} onChange={(e) => { setYearId(Number(e.target.value)); setClasses(null); setConfirmYear(false); }}>
              {years.map((y) => (
                <option key={y.id} value={y.id}>
                  {y.label}
                  {y.active ? " (aktif)" : ""}
                </option>
              ))}
            </select>
          </div>
        </div>
        {year && !year.active ? (
          confirmYear ? (
            <div className="confirm-row" role="group" aria-label={`Konfirmasi aktifkan ${year.label}`}>
              <p>Jadikan {year.label} tahun ajaran aktif? Tabel publik hanya menampilkan tahun aktif.</p>
              <button className="button-primary" type="button" disabled={busy} onClick={() => { setConfirmYear(false); run(async () => { await activateYear(year.id, csrf); return `${year.label} sekarang tahun ajaran aktif.`; }); }}>
                Ya, aktifkan
              </button>
              <button className="button-secondary" type="button" onClick={() => setConfirmYear(false)}>
                Batal
              </button>
            </div>
          ) : (
            <p>
              <button className="button-secondary" type="button" disabled={busy} onClick={() => setConfirmYear(true)}>
                Jadikan {year.label} tahun aktif
              </button>
            </p>
          )
        ) : null}
        <form className="inline-form" onSubmit={onAddYear} noValidate>
          <div className="form-field">
            <label htmlFor="tahun-baru">Tambah tahun ajaran</label>
            <input id="tahun-baru" type="text" inputMode="numeric" placeholder="2027/2028" maxLength={9} value={newYear} onChange={(e) => setNewYear(e.target.value)} />
          </div>
          <div className="form-actions">
            <button className="button-secondary" type="submit" disabled={busy || newYear.trim() === ""}>
              Tambah tahun ajaran
            </button>
          </div>
        </form>
      </section>

      <section className="content-section" aria-labelledby="daftar-kelas">
        <div className="section-heading">
          <h2 id="daftar-kelas">Kelas {year ? year.label : ""}</h2>
        </div>
        {classes === null ? (
          <p className="public-state loading" role="status">
            Memuat kelas…
          </p>
        ) : classes.length === 0 ? (
          <div className="empty-block" role="status">
            <p>Belum ada kelas untuk tahun ajaran ini. Tambahkan kelas pertama di bawah.</p>
          </div>
        ) : (
          <ul className="admin-list">
            {classes.map((c) => (
              <li className="admin-row" key={c.id}>
                <div className="admin-row-main">
                  <h3>
                    {c.code}
                    {c.name ? ` — ${c.name}` : ""}
                  </h3>
                  <p className="admin-row-meta">
                    <ActiveBadge active={c.active} />
                    <span>{c.groupCount} kelompok</span>
                  </p>
                </div>
                {editId === c.id ? (
                  <form className="inline-form" onSubmit={(e) => onSaveEdit(e, c)} noValidate aria-label={`Ubah kelas ${c.code}`}>
                    <ClassFields draft={edit} errors={editErrors} set={setEdit} idPrefix={`ubah-${c.id}`} />
                    <div className="form-actions">
                      <button className="button-primary" type="submit" disabled={busy}>
                        {busy ? "Menyimpan…" : "Simpan kelas"}
                      </button>
                      <button className="button-secondary" type="button" onClick={() => setEditId(null)}>
                        Batal
                      </button>
                    </div>
                  </form>
                ) : confirmId === c.id ? (
                  <div className="confirm-row" role="group" aria-label={`Konfirmasi hapus ${c.code}`}>
                    <p>Hapus kelas {c.code}? Tindakan ini tidak dapat dibatalkan.</p>
                    <button className="button-danger" type="button" disabled={busy} onClick={() => remove(c)}>
                      Ya, hapus kelas
                    </button>
                    <button className="button-secondary" type="button" onClick={() => setConfirmId(null)}>
                      Batal
                    </button>
                  </div>
                ) : (
                  <div className="admin-row-actions">
                    <Link className="button-secondary" href={`/admin/kelompok/?kelas=${c.id}`}>
                      Kelompok<span className="visually-hidden"> kelas {c.code}</span>
                    </Link>
                    <button className="button-secondary" type="button" disabled={busy} onClick={() => { setEditId(c.id); setEdit({ yearId: c.yearId, code: c.code, name: c.name, active: c.active }); setEditErrors({}); }}>
                      Ubah<span className="visually-hidden"> kelas {c.code}</span>
                    </button>
                    <button className="button-secondary" type="button" disabled={busy} onClick={() => toggle(c)}>
                      {c.active ? "Nonaktifkan" : "Aktifkan"}
                      <span className="visually-hidden"> kelas {c.code}</span>
                    </button>
                    {c.groupCount === 0 ? (
                      <button className="button-secondary" type="button" disabled={busy} onClick={() => setConfirmId(c.id)}>
                        Hapus<span className="visually-hidden"> kelas {c.code}</span>
                      </button>
                    ) : null}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
        <p className="field-hint">Kelas yang masih punya kelompok tidak bisa dihapus; nonaktifkan untuk menyembunyikannya dari publik.</p>

        {yearId !== null ? (
          <form className="form-card form-wide inline-form" onSubmit={onAddClass} noValidate aria-labelledby="tambah-kelas">
            <h3 id="tambah-kelas" className="form-actions">
              Tambah kelas
            </h3>
            <ClassFields draft={newClass} errors={newErrors} set={setNewClass} idPrefix="baru" />
            <div className="form-actions">
              <button className="button-primary" type="submit" disabled={busy}>
                Tambah kelas
              </button>
            </div>
          </form>
        ) : null}
      </section>
    </>
  );
}

export function ClassManagerPage() {
  return <AdminPageGate>{(csrf, onExpired) => <Manager csrf={csrf} onExpired={onExpired} />}</AdminPageGate>;
}
