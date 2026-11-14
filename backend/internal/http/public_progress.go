package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

// publicProgress serves the public table: active year, classes, and one safe
// row per active group. Revalidated on every load so teacher edits show up.
func publicProgress(store *progress.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		p, err := store.PublicProgress(r.Context(), progress.PublicFilter{Class: q.Get("class"), Query: q.Get("q"), Status: q.Get("status")})
		if errors.Is(err, progress.ErrInvalid) {
			badQuery(w)
			return
		}
		if err != nil {
			slog.Error("public progress failed", "error", err)
			writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		writeJSON(w, 200, dataEnvelope{p})
	}
}

// publicProgressSummary keeps the Fase 0 summary contract (totalGroups,
// manualUpdate, updatedAt) and adds the lane counts.
func publicProgressSummary(store *progress.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := store.PublicProgress(r.Context(), progress.PublicFilter{})
		if err != nil {
			slog.Error("public summary failed", "error", err)
			writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
			return
		}
		sum, latest := progress.Summarize(p.Groups)
		w.Header().Set("Cache-Control", "no-cache")
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"totalGroups":  sum.Total,
				"none":         sum.None,
				"partial":      sum.Partial,
				"full":         sum.Full,
				"complete":     sum.Complete,
				"manualUpdate": true,
				"updatedAt":    latest,
			},
		})
	}
}
