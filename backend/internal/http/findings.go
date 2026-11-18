package httpapi

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
	"net/http"
	"strconv"
)

type findingRequest struct {
	FileType  string `json:"fileType"`
	Location  string `json:"location"`
	Problem   string `json:"problem"`
	Action    string `json:"action"`
	Priority  string `json:"priority"`
	Resolved  bool   `json:"resolved"`
	UpdatedAt string `json:"updatedAt"`
}

func (f findingRequest) input() progress.FindingInput {
	return progress.FindingInput{FileType: f.FileType, Location: f.Location, Problem: f.Problem, Action: f.Action, Priority: f.Priority}
}
func findingErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, progress.ErrPublished):
		writeError(w, 409, "FINDING_PUBLISHED", "Isi temuan terbit tidak dapat diubah atau dihapus. Tandai selesai setelah diperbaiki.")
	case errors.Is(err, progress.ErrReviewRule):
		writeError(w, 409, "REVIEW_RULE", "Status pemeriksaan tidak sesuai. Perlu Revisi membutuhkan temuan wajib terbit dan terbuka; Disetujui membutuhkan PPT/PDF lengkap tanpa temuan wajib terbuka.")
	default:
		progressErr(w, err)
	}
}
func (a adminProgress) findings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	if r.Method == http.MethodGet {
		fs, err := a.store.Findings(r.Context(), id)
		if err != nil {
			findingErr(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{fs})
		return
	}
	in, ok := decodeJSON[findingRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	f, err := a.store.CreateFinding(r.Context(), id, in.input(), actorOf(r))
	if err != nil {
		findingErr(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/admin/findings/"+strconv.FormatInt(f.ID, 10))
	writeJSON(w, 201, dataEnvelope{f})
}
func (a adminProgress) updateFinding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	in, ok := decodeJSON[findingRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	at, err := parseStamp(in.UpdatedAt)
	if err != nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	f, err := a.store.UpdateFinding(r.Context(), id, at, in.input(), in.Resolved, actorOf(r))
	if err != nil {
		findingErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{f})
}
func (a adminProgress) deleteFinding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	at, err := parseStamp(r.URL.Query().Get("updatedAt"))
	if err != nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	if err = a.store.DeleteFinding(r.Context(), id, at, actorOf(r)); err != nil {
		findingErr(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a adminProgress) publishFindings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	in, ok := decodeJSON[struct {
		Status    progress.Review `json:"status"`
		UpdatedAt string          `json:"updatedAt"`
	}](w, r, maxProgressBody)
	if !ok {
		return
	}
	at, err := parseStamp(in.UpdatedAt)
	if err != nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	if err = a.store.PublishReview(r.Context(), id, at, in.Status, actorOf(r)); err != nil {
		findingErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{map[string]bool{"published": true}})
}
func (a adminProgress) review(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	if r.Method == http.MethodGet {
		v, err := a.store.GetReview(r.Context(), id)
		if err != nil {
			findingErr(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{v})
		return
	}
	in, ok := decodeJSON[struct {
		Status    progress.Review `json:"status"`
		UpdatedAt string          `json:"updatedAt"`
	}](w, r, maxProgressBody)
	if !ok {
		return
	}
	at, err := parseStamp(in.UpdatedAt)
	if err != nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	v, err := a.store.SetReview(r.Context(), id, at, in.Status, actorOf(r))
	if err != nil {
		findingErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{v})
}
func publicGroupDetail(store *progress.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		id, err := parseID(r)
		if err != nil {
			progressErr(w, progress.ErrNotFound)
			return
		}
		d, err := store.PublicDetail(r.Context(), id)
		if err != nil {
			progressErr(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{d})
	}
}
func (a adminProgress) findingRoutes(r chi.Router) {
	r.Get("/groups/{id}/findings", a.findings)
	r.Post("/groups/{id}/findings", a.findings)
	r.Patch("/findings/{id}", a.updateFinding)
	r.Delete("/findings/{id}", a.deleteFinding)
	r.Post("/groups/{id}/publish", a.publishFindings)
	r.Get("/groups/{id}/review-status", a.review)
	r.Patch("/groups/{id}/review-status", a.review)
}
