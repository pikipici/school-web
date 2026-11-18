package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

const maxProgressBody = 8 << 10

type adminProgress struct {
	store *progress.Store
	audit AuditRecorder
}

type yearView struct {
	ID        int64     `json:"id"`
	Label     string    `json:"label"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type classView struct {
	ID         int64     `json:"id"`
	YearID     int64     `json:"yearId"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Active     bool      `json:"active"`
	GroupCount int64     `json:"groupCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// groupView is the admin view. Email is for the teacher only.
type groupView struct {
	ID        int64     `json:"id"`
	ClassID   int64     `json:"classId"`
	TopicID   int64     `json:"topicId"`
	Name      string    `json:"name"`
	Email     *string   `json:"email"`
	SortOrder int32     `json:"sortOrder"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func yearOut(y progress.Year) yearView {
	return yearView{y.ID, y.Label, y.Active, y.CreatedAt, y.UpdatedAt}
}
func classOut(c progress.Class) classView {
	return classView{c.ID, c.YearID, c.Code, c.Name, c.Active, c.GroupCount, c.CreatedAt, c.UpdatedAt}
}
func groupOut(g progress.Group) groupView {
	return groupView{g.ID, g.ClassID, g.TopicID, g.Name, g.Email, g.SortOrder, g.Active, g.CreatedAt, g.UpdatedAt}
}

type yearRequest struct {
	Label string `json:"label"`
}

type classRequest struct {
	YearID    int64  `json:"yearId"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Active    *bool  `json:"active"`
	UpdatedAt string `json:"updatedAt"`
}

type groupRequest struct {
	ClassID   int64  `json:"classId"`
	TopicID   int64  `json:"topicId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	SortOrder int32  `json:"sortOrder"`
	Active    *bool  `json:"active"`
	UpdatedAt string `json:"updatedAt"`
}

// decodeJSON reads one JSON object of type T, rejecting other media types,
// unknown fields, trailing data, and bodies over limit bytes.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, limit int64) (T, bool) {
	var x T
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Gunakan Content-Type application/json.")
		return x, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err = d.Decode(&x); err != nil {
		var big *http.MaxBytesError
		if errors.As(err, &big) {
			writeError(w, 413, "PAYLOAD_TOO_LARGE", "Permintaan terlalu besar.")
		} else {
			writeError(w, 400, "INVALID_REQUEST", "Permintaan tidak valid.")
		}
		return x, false
	}
	if err = d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, 400, "INVALID_REQUEST", "Permintaan tidak valid.")
		return x, false
	}
	return x, true
}

func progressErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, progress.ErrNotFound):
		writeError(w, 404, "NOT_FOUND", "Data tidak ditemukan.")
	case errors.Is(err, progress.ErrConflict):
		writeError(w, 409, "CONFLICT", "Data telah berubah. Muat ulang lalu coba lagi.")
	case errors.Is(err, progress.ErrDuplicate):
		writeError(w, 409, "DUPLICATE", "Nama atau kode sudah dipakai.")
	case errors.Is(err, progress.ErrInUse):
		writeError(w, 409, "IN_USE", "Masih punya data terkait. Nonaktifkan saja.")
	case errors.Is(err, progress.ErrInvalidReference):
		writeError(w, 422, "INVALID_REFERENCE", "Tahun ajaran, kelas, atau topik tidak ada.")
	case errors.Is(err, progress.ErrInvalid):
		writeError(w, 422, "VALIDATION_ERROR", "Isian tidak sesuai aturan.")
	default:
		slog.Error("progress request failed", "error", err)
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
	}
}

func badQuery(w http.ResponseWriter) {
	writeError(w, 400, "VALIDATION_ERROR", "Parameter permintaan tidak valid.")
}

// queryID reads a positive integer query parameter. ok is false when it is
// present but malformed; present is false when it is absent.
func queryID(r *http.Request, name string) (id int64, present, ok bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, false, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, true, false
	}
	return id, true, true
}

func parseStamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, progress.ErrInvalid
	}
	return t, nil
}

func (a adminProgress) record(r *http.Request, action string, id int64, title string) {
	if a.audit == nil {
		return
	}
	s, ok := AdminSessionFromContext(r.Context())
	if !ok {
		return
	}
	uid := s.UserID
	a.audit.Record(r.Context(), AuditEvent{Action: "progress." + action, Outcome: "success", UserID: &uid, IP: clientIP(r), Detail: map[string]string{"entityId": strconv.FormatInt(id, 10), "title": title}})
}

// ---- academic years ----------------------------------------------------------

func (a adminProgress) listYears(w http.ResponseWriter, r *http.Request) {
	ys, err := a.store.Years(r.Context())
	if err != nil {
		progressErr(w, err)
		return
	}
	out := make([]yearView, 0, len(ys))
	for _, y := range ys {
		out = append(out, yearOut(y))
	}
	writeJSON(w, 200, dataEnvelope{out})
}

func (a adminProgress) createYear(w http.ResponseWriter, r *http.Request) {
	q, ok := decodeJSON[yearRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	y, err := a.store.CreateYear(r.Context(), q.Label)
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "year_created", y.ID, y.Label)
	w.Header().Set("Location", "/api/v1/admin/academic-years/"+strconv.FormatInt(y.ID, 10))
	writeJSON(w, 201, dataEnvelope{yearOut(y)})
}

func (a adminProgress) activateYear(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	if err := a.store.ActivateYear(r.Context(), id); err != nil {
		progressErr(w, err)
		return
	}
	label := ""
	if ys, err := a.store.Years(r.Context()); err == nil {
		for _, y := range ys {
			if y.ID == id {
				label = y.Label
			}
		}
	}
	a.record(r, "year_activated", id, label)
	w.WriteHeader(204)
}

// ---- classes -----------------------------------------------------------------

func (a adminProgress) listClasses(w http.ResponseWriter, r *http.Request) {
	year, present, ok := queryID(r, "year")
	if !ok {
		badQuery(w)
		return
	}
	if !present {
		ys, err := a.store.Years(r.Context())
		if err != nil {
			progressErr(w, err)
			return
		}
		for _, y := range ys {
			if y.Active {
				year = y.ID
			}
		}
	}
	cs, err := a.store.Classes(r.Context(), year)
	if err != nil {
		progressErr(w, err)
		return
	}
	out := make([]classView, 0, len(cs))
	for _, c := range cs {
		out = append(out, classOut(c))
	}
	writeJSON(w, 200, dataEnvelope{out})
}

func (a adminProgress) getClass(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	c, err := a.store.Class(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{classOut(c)})
}

func (a adminProgress) createClass(w http.ResponseWriter, r *http.Request) {
	q, ok := decodeJSON[classRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	c, err := a.store.CreateClass(r.Context(), progress.ClassInput{YearID: q.YearID, Code: q.Code, Name: q.Name})
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "class_created", c.ID, c.Code)
	w.Header().Set("Location", "/api/v1/admin/classes/"+strconv.FormatInt(c.ID, 10))
	writeJSON(w, 201, dataEnvelope{classOut(c)})
}

func (a adminProgress) updateClass(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	q, ok := decodeJSON[classRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	expected, err := parseStamp(q.UpdatedAt)
	if err != nil || q.Active == nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	c, err := a.store.UpdateClass(r.Context(), id, expected, progress.ClassInput{YearID: q.YearID, Code: q.Code, Name: q.Name, Active: *q.Active})
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "class_updated", c.ID, c.Code)
	writeJSON(w, 200, dataEnvelope{classOut(c)})
}

func (a adminProgress) deleteClass(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	code, err := a.store.DeleteClass(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "class_deleted", id, code)
	w.WriteHeader(204)
}

// ---- groups ------------------------------------------------------------------

func (a adminProgress) listGroups(w http.ResponseWriter, r *http.Request) {
	class, present, ok := queryID(r, "class")
	if !ok || !present {
		badQuery(w)
		return
	}
	gs, err := a.store.Groups(r.Context(), class)
	if err != nil {
		progressErr(w, err)
		return
	}
	out := make([]groupView, 0, len(gs))
	for _, g := range gs {
		out = append(out, groupOut(g))
	}
	writeJSON(w, 200, dataEnvelope{out})
}

func (a adminProgress) getGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	g, err := a.store.Group(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{groupOut(g)})
}

func (a adminProgress) createGroup(w http.ResponseWriter, r *http.Request) {
	q, ok := decodeJSON[groupRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	g, err := a.store.CreateGroup(r.Context(), progress.GroupInput{ClassID: q.ClassID, TopicID: q.TopicID, Name: q.Name, Email: q.Email, SortOrder: q.SortOrder})
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "group_created", g.ID, g.Name)
	w.Header().Set("Location", "/api/v1/admin/groups/"+strconv.FormatInt(g.ID, 10))
	writeJSON(w, 201, dataEnvelope{groupOut(g)})
}

func (a adminProgress) updateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	q, ok := decodeJSON[groupRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	expected, err := parseStamp(q.UpdatedAt)
	if err != nil || q.Active == nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	g, err := a.store.UpdateGroup(r.Context(), id, expected, progress.GroupInput{ClassID: q.ClassID, TopicID: q.TopicID, Name: q.Name, Email: q.Email, SortOrder: q.SortOrder, Active: *q.Active})
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "group_updated", g.ID, g.Name)
	writeJSON(w, 200, dataEnvelope{groupOut(g)})
}

func (a adminProgress) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	name, err := a.store.DeleteGroup(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	a.record(r, "group_deleted", id, name)
	w.WriteHeader(204)
}

// routes mounts the admin progress API on r, which must already require a
// valid admin session (and CSRF + Origin for unsafe methods).
func (a adminProgress) routes(r chi.Router) {
	r.Get("/academic-years", a.listYears)
	r.Post("/academic-years", a.createYear)
	r.Post("/academic-years/{id}/activate", a.activateYear)
	r.Get("/classes", a.listClasses)
	r.Post("/classes", a.createClass)
	r.Get("/classes/{id}", a.getClass)
	r.Patch("/classes/{id}", a.updateClass)
	r.Delete("/classes/{id}", a.deleteClass)
	r.Get("/groups", a.listGroups)
	r.Post("/groups", a.createGroup)
	r.Get("/groups/{id}", a.getGroup)
	r.Patch("/groups/{id}", a.updateGroup)
	r.Delete("/groups/{id}", a.deleteGroup)
	a.submissionRoutes(r)
	a.findingRoutes(r)
}
