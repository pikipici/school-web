"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import {
  AdminContentError,
  LIMITS,
  createItem,
  draftFrom,
  emptyDraft,
  getItem,
  messageForContent,
  updateItem,
  validateDraft,
  type AdminItem,
  type ContentKind,
  type Draft,
  type DraftErrors
} from "@/lib/api-client/admin-content";
import { getTopics, type Topic } from "@/lib/api-client/content";
import { formatWIB, validContentID } from "@/lib/format";

import { useAdminSession } from "./useAdminSession";

const ADMIN_HOME = "/admin/";

const LABELS: Record<ContentKind, { create: string; edit: string; text: string; textHint: string; save: string }> = {
  announcement: {
    create: "Tulis pengumuman",
    edit: "Ubah pengumuman",
    text: "Isi pengumuman",
    textHint: "Teks biasa. Baris kosong memisahkan paragraf. Alamat https:// menjadi tautan.",
    save: "Simpan pengumuman"
  },
  material: {
    create: "Tambah materi",
    edit: "Ubah materi",
    text: "Deskripsi materi",
    textHint: "Teks biasa. Baris kosong memisahkan paragraf. Alamat https:// menjadi tautan.",
    save: "Simpan materi"
  }
};

const chars = (s: string) => [...s].length;

type LoadState =
  | { status: "loading" }
  | { status: "error"; error: AdminContentError }
  | { status: "ready"; item: AdminItem | null };

function Editor({ kind, id, csrf, onExpired }: { kind: ContentKind; id: number | null; csrf: string; onExpired: () => void }) {
  const router = useRouter();
  const label = LABELS[kind];
  const [load, setLoad] = useState<LoadState>(id === null ? { status: "ready", item: null } : { status: "loading" });
  const [tick, setTick] = useState(0);
  const [draft, setDraft] = useState<Draft>(() => emptyDraft(kind));
  const [errors, setErrors] = useState<DraftErrors>({});
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<AdminContentError | null>(null);
  const [topics, setTopics] = useState<Topic[] | null>(null);
  const [topicsFailed, setTopicsFailed] = useState(false);
  const errorRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (id === null) return;
    const controller = new AbortController();
    getItem(kind, id, controller.signal).then(
      (item) => {
        setLoad({ status: "ready", item });
        setDraft(draftFrom(item));
        setSaveError(null);
      },
      (caught: unknown) => {
        if (caught instanceof DOMException && caught.name === "AbortError") return;
        const error = caught instanceof AdminContentError ? caught : new AdminContentError("server");
        if (error.kind === "unauthenticated") onExpired();
        else setLoad({ status: "error", error });
      }
    );
    return () => controller.abort();
  }, [kind, id, tick, onExpired]);

  useEffect(() => {
    if (kind !== "material") return;
    const controller = new AbortController();
    getTopics(controller.signal).then(setTopics, (caught: unknown) => {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setTopicsFailed(true);
    });
    return () => controller.abort();
  }, [kind]);

  useEffect(() => {
    if (saveError || Object.keys(errors).length > 0) errorRef.current?.focus();
  }, [saveError, errors]);

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) => setDraft((d) => ({ ...d, [key]: value }));

  function reloadLatest() {
    setLoad({ status: "loading" });
    setTick((n) => n + 1);
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (saving || load.status !== "ready") return;
    const found = validateDraft(kind, draft);
    setErrors(found);
    setSaveError(null);
    if (Object.keys(found).length > 0) return;

    setSaving(true);
    try {
      if (load.item) await updateItem(kind, load.item.id, draft, load.item.updatedAt, csrf);
      else await createItem(kind, draft, csrf);
      router.replace(ADMIN_HOME);
    } catch (caught) {
      const error = caught instanceof AdminContentError ? caught : new AdminContentError("server");
      if (error.kind === "unauthenticated") {
        onExpired();
        return;
      }
      setSaveError(error);
      setSaving(false);
    }
  }

  if (load.status === "loading") {
    return (
      <p className="public-state loading" role="status">
        Memuat konten…
      </p>
    );
  }
  if (load.status === "error") {
    return (
      <div className="public-state error" role="alert">
        <p>{messageForContent(load.error.kind)}</p>
        {load.error.kind !== "not_found" ? (
          <button className="retry-button" type="button" onClick={reloadLatest}>
            Coba lagi
          </button>
        ) : null}
        <p>
          <Link href={ADMIN_HOME}>Kembali ke dasbor</Link>
        </p>
      </div>
    );
  }

  const textMax = LIMITS[kind].text;
  const titleCount = chars(draft.title.trim());
  const textCount = chars(draft.text);
  const errorList = Object.values(errors);

  return (
    <>
      <h1 className="page-title">{load.item ? label.edit : label.create}</h1>
      <p>
        <Link href={ADMIN_HOME}>Kembali ke dasbor tanpa menyimpan</Link>
      </p>
      {load.item ? (
        <p className="section-note">
          Terakhir diubah {formatWIB(load.item.updatedAt)}
          {load.item.publishedAt ? `. Pertama terbit ${formatWIB(load.item.publishedAt)}.` : "."}
        </p>
      ) : null}

      <form className="form-card form-wide" onSubmit={onSubmit} noValidate aria-busy={saving}>
        <div ref={errorRef} tabIndex={-1} aria-live="assertive">
          {saveError ? (
            <div className="form-error" role="alert">
              <span className="notice-icon" aria-hidden="true">!</span>
              <div>
                <p>{messageForContent(saveError.kind)} Isian Anda masih ada di formulir ini.</p>
                {saveError.kind === "conflict" ? (
                  <button className="button-secondary" type="button" onClick={reloadLatest}>
                    Muat data terbaru (isian ini hilang)
                  </button>
                ) : null}
              </div>
            </div>
          ) : errorList.length > 0 ? (
            <div className="form-error" role="alert">
              <span className="notice-icon" aria-hidden="true">!</span>
              <div>
                <p>Periksa isian berikut:</p>
                <ul>
                  {errorList.map((e) => (
                    <li key={e}>{e}</li>
                  ))}
                </ul>
              </div>
            </div>
          ) : null}
        </div>

        <div className="form-field">
          <label htmlFor="judul">Judul (wajib)</label>
          <input
            id="judul"
            type="text"
            required
            value={draft.title}
            aria-invalid={errors.title ? true : undefined}
            aria-describedby="judul-hitung"
            onChange={(e) => set("title", e.target.value)}
          />
          <p id="judul-hitung" className={titleCount > LIMITS.title ? "field-count over" : "field-count"}>
            {titleCount} / {LIMITS.title} karakter
          </p>
        </div>

        <div className="form-field">
          <label htmlFor="isi">{label.text} (opsional)</label>
          <textarea
            id="isi"
            rows={kind === "announcement" ? 8 : 12}
            value={draft.text}
            aria-invalid={errors.text ? true : undefined}
            aria-describedby="isi-petunjuk isi-hitung"
            onChange={(e) => set("text", e.target.value)}
          />
          <p id="isi-petunjuk" className="field-hint">
            {label.textHint}
          </p>
          <p id="isi-hitung" className={textCount > textMax ? "field-count over" : "field-count"}>
            {textCount.toLocaleString("id-ID")} / {textMax.toLocaleString("id-ID")} karakter
          </p>
        </div>

        {kind === "material" ? (
          <>
            <div className="form-field">
              <label htmlFor="topik">Topik</label>
              <select
                id="topik"
                value={draft.topicId === null ? "" : String(draft.topicId)}
                aria-invalid={errors.topicId ? true : undefined}
                disabled={!topics && !topicsFailed}
                onChange={(e) => set("topicId", e.target.value === "" ? null : Number(e.target.value))}
              >
                <option value="">Umum (tanpa topik)</option>
                {(topics ?? []).map((t) => (
                  <option key={t.id} value={String(t.id)}>
                    {t.code} — {t.title}
                  </option>
                ))}
              </select>
              {topicsFailed ? (
                <p className="field-hint" role="status">
                  Daftar topik gagal dimuat. Materi tetap bisa disimpan sebagai Umum.
                </p>
              ) : null}
            </div>
            <div className="form-field">
              <label htmlFor="urutan">Urutan tampil</label>
              <input
                id="urutan"
                type="number"
                inputMode="numeric"
                min={0}
                max={LIMITS.sortOrderMax}
                step={1}
                value={Number.isNaN(draft.sortOrder) ? "" : String(draft.sortOrder)}
                aria-invalid={errors.sortOrder ? true : undefined}
                aria-describedby="urutan-petunjuk"
                onChange={(e) => set("sortOrder", e.target.value === "" ? Number.NaN : Number(e.target.value))}
              />
              <p id="urutan-petunjuk" className="field-hint">
                Angka kecil tampil lebih dulu di dalam topiknya.
              </p>
            </div>
          </>
        ) : (
          <div className="form-check">
            <input id="sematkan" type="checkbox" checked={draft.pinned} onChange={(e) => set("pinned", e.target.checked)} />
            <label htmlFor="sematkan">Sematkan di atas daftar pengumuman</label>
          </div>
        )}

        <fieldset className="form-field">
          <legend>Status</legend>
          <div className="form-check">
            <input id="status-draf" type="radio" name="status" checked={draft.status === "draft"} onChange={() => set("status", "draft")} />
            <label htmlFor="status-draf">Draf — hanya terlihat di dasbor</label>
          </div>
          <div className="form-check">
            <input id="status-terbit" type="radio" name="status" checked={draft.status === "published"} onChange={() => set("status", "published")} />
            <label htmlFor="status-terbit">Terbit — langsung terlihat oleh pengunjung</label>
          </div>
        </fieldset>

        <button className="button-primary" type="submit" disabled={saving}>
          {saving ? "Menyimpan…" : label.save}
        </button>
      </form>
    </>
  );
}

export function ContentEditorPage({ kind }: { kind: ContentKind }) {
  const { state, retry, expired } = useAdminSession();
  const rawId = useSearchParams().get("id");
  const id = rawId === null ? null : validContentID(rawId);

  if (state.status === "loading" || state.status === "redirecting") {
    return (
      <p className="public-state loading" role="status">
        {state.status === "loading" ? "Memeriksa sesi masuk…" : "Mengalihkan ke halaman masuk…"}
      </p>
    );
  }
  if (state.status === "error") {
    return (
      <div className="public-state error" role="alert">
        <p>Sesi belum dapat diperiksa. Periksa koneksi Anda, lalu coba lagi.</p>
        <button className="retry-button" type="button" onClick={retry}>
          Coba lagi
        </button>
      </div>
    );
  }
  if (rawId !== null && id === null) {
    return (
      <div className="public-state error" role="alert">
        <p>ID konten tidak valid.</p>
        <Link href={ADMIN_HOME}>Kembali ke dasbor</Link>
      </div>
    );
  }
  return <Editor kind={kind} id={id} csrf={state.session.csrfToken} onExpired={expired} />;
}
