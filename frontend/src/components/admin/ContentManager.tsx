"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";

import {
  AdminContentError,
  deleteItem,
  draftFrom,
  listItems,
  messageForContent,
  setStatus,
  type AdminItem,
  type ContentKind,
  type PageMeta
} from "@/lib/api-client/admin-content";
import { getTopics, type Topic } from "@/lib/api-client/content";
import { formatWIB } from "@/lib/format";

const PAGE_SIZE = 20;

export const EDIT_PATH: Record<ContentKind, string> = {
  announcement: "/admin/pengumuman/ubah/",
  material: "/admin/materi/ubah/"
};

const LABELS: Record<ContentKind, { heading: string; create: string; empty: string; noun: string }> = {
  announcement: {
    heading: "Pengumuman",
    create: "Tulis pengumuman",
    empty: "Belum ada pengumuman. Tulis pengumuman pertama untuk siswa.",
    noun: "pengumuman"
  },
  material: {
    heading: "Materi",
    create: "Tambah materi",
    empty: "Belum ada materi. Tambahkan materi pertama untuk siswa.",
    noun: "materi"
  }
};

export function StatusBadge({ item }: { item: AdminItem }) {
  if (item.status === "published") {
    return (
      <span className="status-badge status-published">
        <span aria-hidden="true">✓</span> Terbit
      </span>
    );
  }
  return (
    <span className="status-badge status-draft">
      <span aria-hidden="true">○</span> Draf
    </span>
  );
}

type ListState =
  | { status: "loading" }
  | { status: "error"; error: AdminContentError }
  | { status: "ready"; items: AdminItem[]; meta: PageMeta };

function ContentSection({
  kind,
  csrf,
  topics,
  onExpired
}: {
  kind: ContentKind;
  csrf: string;
  topics: Topic[] | null;
  onExpired: () => void;
}) {
  const label = LABELS[kind];
  const [page, setPage] = useState(1);
  const [tick, setTick] = useState(0);
  const [state, setState] = useState<ListState>({ status: "loading" });
  const [busyId, setBusyId] = useState<number | null>(null);
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [notice, setNotice] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const controller = new AbortController();
    listItems(kind, { page, limit: PAGE_SIZE }, controller.signal).then(
      ({ items, meta }) => setState({ status: "ready", items, meta }),
      (caught: unknown) => {
        if (caught instanceof DOMException && caught.name === "AbortError") return;
        const error = caught instanceof AdminContentError ? caught : new AdminContentError("server");
        if (error.kind === "unauthenticated") onExpired();
        else setState({ status: "error", error });
      }
    );
    return () => controller.abort();
  }, [kind, page, tick, onExpired]);

  useEffect(() => {
    if (confirmId !== null) cancelRef.current?.focus();
  }, [confirmId]);

  const reload = () => {
    setState({ status: "loading" });
    setTick((n) => n + 1);
  };

  const fail = (caught: unknown) => {
    const error = caught instanceof AdminContentError ? caught : new AdminContentError("server");
    if (error.kind === "unauthenticated") {
      onExpired();
      return;
    }
    setNotice({ tone: "error", text: messageForContent(error.kind) });
    if (error.kind === "conflict" || error.kind === "not_found") reload();
  };

  async function toggle(item: AdminItem) {
    const next = item.status === "published" ? "draft" : "published";
    setBusyId(item.id);
    setNotice(null);
    try {
      await setStatus({ ...draftFrom(item), kind, id: item.id, updatedAt: item.updatedAt }, next, csrf);
      setNotice({ tone: "ok", text: next === "published" ? `“${item.title}” sudah terbit.` : `“${item.title}” ditarik menjadi draf.` });
      reload();
    } catch (caught) {
      fail(caught);
    } finally {
      setBusyId(null);
    }
  }

  async function remove(item: AdminItem) {
    setBusyId(item.id);
    setNotice(null);
    try {
      await deleteItem(kind, item.id, csrf);
      setConfirmId(null);
      setNotice({ tone: "ok", text: `“${item.title}” sudah dihapus.` });
      if (state.status === "ready" && state.items.length === 1 && page > 1) setPage(page - 1);
      else reload();
    } catch (caught) {
      setConfirmId(null);
      fail(caught);
    } finally {
      setBusyId(null);
    }
  }

  const topicName = (id: number | null) => {
    if (id === null) return "Umum";
    return topics?.find((t) => t.id === id)?.title ?? "Topik tidak dikenal";
  };

  return (
    <section className="content-section" aria-labelledby={`kelola-${kind}`}>
      <div className="section-heading">
        <h2 id={`kelola-${kind}`}>{label.heading}</h2>
        <Link className="button-primary" href={EDIT_PATH[kind]}>
          {label.create}
        </Link>
      </div>

      <div aria-live="polite">
        {notice ? (
          <p className={notice.tone === "ok" ? "form-success" : "form-error"} role={notice.tone === "error" ? "alert" : "status"}>
            <span aria-hidden="true">{notice.tone === "ok" ? "✓ " : "! "}</span>
            {notice.text}
          </p>
        ) : null}
      </div>

      {state.status === "loading" ? (
        <p className="public-state loading" role="status">
          Memuat daftar {label.noun}…
        </p>
      ) : state.status === "error" ? (
        <div className="public-state error" role="alert">
          <p>{messageForContent(state.error.kind)}</p>
          <button className="retry-button" type="button" onClick={reload}>
            Coba lagi
          </button>
        </div>
      ) : state.items.length === 0 ? (
        <div className="empty-block" role="status">
          <p>{label.empty}</p>
        </div>
      ) : (
        <>
          <ul className="admin-list">
            {state.items.map((item) => (
              <li key={item.id} className="admin-row">
                <div className="admin-row-main">
                  <h3>{item.title}</h3>
                  <p className="admin-row-meta">
                    <StatusBadge item={item} />
                    {kind === "announcement" && item.pinned ? (
                      <span className="status-badge status-pinned">
                        <span aria-hidden="true">▲</span> Disematkan
                      </span>
                    ) : null}
                    {kind === "material" ? <span>Topik: {topicName(item.topicId)}</span> : null}
                    <span>
                      {item.publishedAt ? `Terbit ${formatWIB(item.publishedAt)}` : `Diubah ${formatWIB(item.updatedAt)}`}
                    </span>
                  </p>
                </div>

                {confirmId === item.id ? (
                  <div className="confirm-row" role="group" aria-label={`Konfirmasi hapus ${item.title}`}>
                    <p>
                      Hapus “{item.title}” secara permanen? Tindakan ini tidak dapat dibatalkan.
                    </p>
                    <button className="button-danger" type="button" disabled={busyId === item.id} onClick={() => remove(item)}>
                      {busyId === item.id ? "Menghapus…" : "Ya, hapus permanen"}
                    </button>
                    <button ref={cancelRef} className="button-secondary" type="button" disabled={busyId === item.id} onClick={() => setConfirmId(null)}>
                      Batal
                    </button>
                  </div>
                ) : (
                  <div className="admin-row-actions">
                    <Link className="button-secondary" href={`${EDIT_PATH[kind]}?id=${item.id}`}>
                      Ubah<span className="visually-hidden"> {item.title}</span>
                    </Link>
                    <button className="button-secondary" type="button" disabled={busyId !== null} onClick={() => toggle(item)}>
                      {busyId === item.id ? "Menyimpan…" : item.status === "published" ? "Tarik ke draf" : "Terbitkan"}
                      <span className="visually-hidden"> {item.title}</span>
                    </button>
                    <button className="button-secondary" type="button" disabled={busyId !== null} onClick={() => setConfirmId(item.id)}>
                      Hapus<span className="visually-hidden"> {item.title}</span>
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
          {state.meta.totalPages > 1 ? (
            <nav className="pagination" aria-label={`Halaman daftar ${label.noun}`}>
              <button type="button" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                Sebelumnya
              </button>
              <span>
                Halaman {state.meta.page} dari {state.meta.totalPages}
              </span>
              <button type="button" disabled={page >= state.meta.totalPages} onClick={() => setPage(page + 1)}>
                Berikutnya
              </button>
            </nav>
          ) : null}
        </>
      )}
    </section>
  );
}

export function ContentManager({ csrf, onExpired }: { csrf: string; onExpired: () => void }) {
  const [topics, setTopics] = useState<Topic[] | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    getTopics(controller.signal).then(setTopics, () => undefined);
    return () => controller.abort();
  }, []);

  const expired = useCallback(() => onExpired(), [onExpired]);

  return (
    <>
      <ContentSection kind="announcement" csrf={csrf} topics={topics} onExpired={expired} />
      <ContentSection kind="material" csrf={csrf} topics={topics} onExpired={expired} />
    </>
  );
}
