"use client";

import { useEffect, useMemo, useState } from "react";

import {
  NOTEBOOK_LABEL,
  REVIEW_LABEL,
  SUBMISSION_LABEL,
  getPublicProgress,
  type PublicGroup,
  type PublicProgress,
  type StatusFilter
} from "@/lib/api-client/progress";
import { formatWIB } from "@/lib/format";

type Load = { status: "loading" } | { status: "error" } | { status: "ready"; data: PublicProgress };

const SUBMISSION_ICON = { BELUM: "○", SEBAGIAN: "◐", LENGKAP: "●" } as const;
const SUBMISSION_TONE = { BELUM: "lane-none", SEBAGIAN: "lane-partial", LENGKAP: "lane-done" } as const;

const STATUS_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: "", label: "Semua status" },
  { value: "BELUM", label: SUBMISSION_LABEL.BELUM },
  { value: "SEBAGIAN", label: SUBMISSION_LABEL.SEBAGIAN },
  { value: "LENGKAP", label: SUBMISSION_LABEL.LENGKAP },
  { value: "TUNTAS", label: "Tuntas" }
];

function matches(g: PublicGroup, cls: string, status: StatusFilter, q: string): boolean {
  if (cls && g.classCode.toLowerCase() !== cls.toLowerCase()) return false;
  if (status === "TUNTAS" ? !g.complete : status && g.submission !== status) return false;
  if (q && !g.name.toLowerCase().includes(q.toLowerCase())) return false;
  return true;
}

function SubmissionCell({ g }: { g: PublicGroup }) {
  return (
    <span className={`lane-chip ${SUBMISSION_TONE[g.submission]}`}>
      <span aria-hidden="true">{SUBMISSION_ICON[g.submission]}</span> {SUBMISSION_LABEL[g.submission]}
    </span>
  );
}

function ReviewCell({ g }: { g: PublicGroup }) {
  const done = g.review === "DISETUJUI";
  const needs = g.review === "PERLU_REVISI";
  return (
    <span className={`lane-chip ${done ? "lane-done" : needs ? "lane-attention" : "lane-none"}`}>
      <span aria-hidden="true">{done ? "✓" : needs ? "!" : "○"}</span> {REVIEW_LABEL[g.review]}
    </span>
  );
}

function NotebookCell({ g }: { g: PublicGroup }) {
  const done = g.notebook === "SUDAH_DITERIMA";
  return (
    <span className={`lane-chip ${done ? "lane-done" : "lane-none"}`}>
      <span aria-hidden="true">{done ? "✓" : "○"}</span> {NOTEBOOK_LABEL[g.notebook]}
    </span>
  );
}

export function ProgressTable() {
  const [load, setLoad] = useState<Load>({ status: "loading" });
  const [tick, setTick] = useState(0);
  const [cls, setCls] = useState("");
  const [status, setStatus] = useState<StatusFilter>("");
  const [q, setQ] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    getPublicProgress({}, controller.signal).then(
      (data) => setLoad({ status: "ready", data }),
      (e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setLoad({ status: "error" });
      }
    );
    return () => controller.abort();
  }, [tick]);

  const rows = useMemo(
    () => (load.status === "ready" ? load.data.groups.filter((g) => matches(g, cls, status, q.trim())) : []),
    [load, cls, status, q]
  );

  if (load.status === "loading") {
    return (
      <p className="public-state loading" role="status">
        Memuat progres kelompok…
      </p>
    );
  }
  if (load.status === "error") {
    return (
      <div className="public-state error" role="alert">
        <p>Progres kelompok tidak dapat dimuat. Silakan coba lagi.</p>
        <button className="retry-button" type="button" onClick={() => { setLoad({ status: "loading" }); setTick((n) => n + 1); }}>
          Coba lagi
        </button>
      </div>
    );
  }

  const { data } = load;
  const all = data.groups;
  const count = (s: PublicGroup["submission"]) => all.filter((g) => g.submission === s).length;
  const latest = all.reduce<string | null>((m, g) => (m === null || g.updatedAt > m ? g.updatedAt : m), null);

  if (all.length === 0) {
    return (
      <div className="empty-block" role="status">
        <p>Belum ada kelompok yang didaftarkan guru{data.year ? ` untuk tahun ajaran ${data.year}` : ""}.</p>
      </div>
    );
  }

  return (
    <>
      <p className="progress-counts">
        <strong>{all.length} kelompok</strong> pada tahun ajaran {data.year}: {count("BELUM")} {SUBMISSION_LABEL.BELUM.toLowerCase()},{" "}
        {count("SEBAGIAN")} {SUBMISSION_LABEL.SEBAGIAN.toLowerCase()}, {count("LENGKAP")} {SUBMISSION_LABEL.LENGKAP.toLowerCase()},{" "}
        {all.filter((g) => g.complete).length} tuntas.
        {latest ? <> Terakhir diubah {formatWIB(latest)}.</> : null}
      </p>

      <form className="progress-filters" role="search" aria-label="Saring kelompok" onSubmit={(e) => e.preventDefault()}>
        <div className="filter">
          <label htmlFor="saring-kelas">Kelas</label>
          <select id="saring-kelas" value={cls} onChange={(e) => setCls(e.target.value)}>
            <option value="">Semua kelas</option>
            {data.classes.map((c) => (
              <option key={c.code} value={c.code}>
                {c.name ? `${c.code} — ${c.name}` : c.code}
              </option>
            ))}
          </select>
        </div>
        <div className="filter">
          <label htmlFor="saring-status">Pengumpulan</label>
          <select id="saring-status" value={status} onChange={(e) => setStatus(e.target.value as StatusFilter)}>
            {STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div className="filter">
          <label htmlFor="cari-kelompok">Cari nama kelompok</label>
          <input id="cari-kelompok" type="search" maxLength={60} value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
      </form>

      <p className="visually-hidden" role="status" aria-live="polite">
        {rows.length} dari {all.length} kelompok ditampilkan.
      </p>

      {rows.length === 0 ? (
        <div className="empty-block" role="status">
          <p>Tidak ada kelompok untuk pilihan ini.</p>
        </div>
      ) : (
        <table className="progress-table">
          <caption className="visually-hidden">Progres tugas kelompok, {rows.length} kelompok</caption>
          <thead>
            <tr>
              <th scope="col">Kelompok</th>
              <th scope="col">Kelas</th>
              <th scope="col">Topik</th>
              <th scope="col">Pengumpulan</th>
              <th scope="col">Pemeriksaan</th>
              <th scope="col">Buku Tulis</th>
              <th scope="col">Terakhir Diperbarui</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((g) => (
              <tr key={`${g.classCode}/${g.name}`}>
                <th scope="row">
                  {g.name}
                  {g.complete ? (
                    <span className="lane-chip lane-done tuntas">
                      <span aria-hidden="true">★</span> Tuntas
                    </span>
                  ) : null}
                </th>
                <td data-label="Kelas">{g.classCode}</td>
                <td data-label="Topik">
                  <span>
                    <span className="topic-code">{g.topic.code}</span> {g.topic.title}
                  </span>
                </td>
                <td data-label="Pengumpulan">
                  <SubmissionCell g={g} />
                </td>
                <td data-label="Pemeriksaan">
                  <ReviewCell g={g} />
                </td>
                <td data-label="Buku Tulis">
                  <NotebookCell g={g} />
                </td>
                <td data-label="Terakhir Diperbarui">
                  <time dateTime={g.updatedAt}>{formatWIB(g.updatedAt)}</time>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
