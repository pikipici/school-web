package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/content"
)

const maxContentBody = 32 << 10

type adminContent struct {
	store content.AdminStore
	audit AuditRecorder
}
type contentRequest struct {
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	Description string         `json:"description"`
	Status      content.Status `json:"status"`
	Pinned      bool           `json:"pinned"`
	TopicID     *int64         `json:"topicId"`
	SortOrder   int32          `json:"sortOrder"`
	UpdatedAt   string         `json:"updatedAt"`
}
type adminItem struct {
	ID          int64          `json:"id"`
	TopicID     *int64         `json:"topicId,omitempty"`
	Title       string         `json:"title"`
	Body        string         `json:"body,omitempty"`
	Description string         `json:"description,omitempty"`
	Status      content.Status `json:"status"`
	Pinned      bool           `json:"pinned,omitempty"`
	SortOrder   int32          `json:"sortOrder,omitempty"`
	PublishedAt *time.Time     `json:"publishedAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func view(v content.Item, kind content.Kind) adminItem {
	x := adminItem{ID: v.ID, TopicID: v.TopicID, Title: v.Title, Status: v.Status, Pinned: v.Pinned, SortOrder: v.SortOrder, PublishedAt: v.PublishedAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
	if kind == content.Announcement {
		x.Body = v.Body
	} else {
		x.Description = v.Body
	}
	return x
}
func decodeContent(w http.ResponseWriter, r *http.Request) (contentRequest, bool) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Gunakan Content-Type application/json.")
		return contentRequest{}, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxContentBody)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var x contentRequest
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
func adminErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrNotFound):
		writeError(w, 404, "NOT_FOUND", "Konten tidak ditemukan.")
	case errors.Is(err, content.ErrConflict):
		writeError(w, 409, "CONFLICT", "Konten telah berubah. Muat ulang lalu coba lagi.")
	case errors.Is(err, content.ErrInvalid), errors.Is(err, content.ErrInvalidTopic):
		writeError(w, 422, "VALIDATION_ERROR", "Data konten tidak valid.")
	default:
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
	}
}
func (a adminContent) list(kind content.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, m, e := parsePage(r)
		if e != nil {
			adminErr(w, e)
			return
		}
		var xs []content.Item
		var n int64
		if kind == content.Announcement {
			xs, n, e = a.store.AdminAnnouncements(r.Context(), p)
		} else {
			xs, n, e = a.store.AdminMaterials(r.Context(), p)
		}
		if e != nil {
			adminErr(w, e)
			return
		}
		out := make([]adminItem, 0, len(xs))
		for _, x := range xs {
			out = append(out, view(x, kind))
		}
		writeJSON(w, 200, listEnvelope{out, listMeta(m, n)})
	}
}
func (a adminContent) get(kind content.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, e := parseID(r)
		if e != nil {
			adminErr(w, e)
			return
		}
		var x content.Item
		if kind == content.Announcement {
			x, e = a.store.AdminAnnouncement(r.Context(), id)
		} else {
			x, e = a.store.AdminMaterial(r.Context(), id)
		}
		if e != nil {
			adminErr(w, e)
			return
		}
		writeJSON(w, 200, dataEnvelope{view(x, kind)})
	}
}
func (a adminContent) create(kind content.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, ok := decodeContent(w, r)
		if !ok {
			return
		}
		in := content.Input{Title: q.Title, Body: q.Body, Status: q.Status, Pinned: q.Pinned, TopicID: q.TopicID, SortOrder: q.SortOrder}
		if kind == content.Material {
			in.Body = q.Description
		}
		var x content.Item
		var e error
		if kind == content.Announcement {
			x, e = a.store.CreateAnnouncement(r.Context(), in)
		} else {
			x, e = a.store.CreateMaterial(r.Context(), in)
		}
		if e != nil {
			adminErr(w, e)
			return
		}
		a.record(r, kind, "created", x.ID, x.Title)
		w.Header().Set("Location", r.URL.Path+"/"+strconv.FormatInt(x.ID, 10))
		writeJSON(w, 201, dataEnvelope{view(x, kind)})
	}
}
func (a adminContent) update(kind content.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, e := parseID(r)
		if e != nil {
			adminErr(w, e)
			return
		}
		q, ok := decodeContent(w, r)
		if !ok {
			return
		}
		expected, e := time.Parse(time.RFC3339Nano, q.UpdatedAt)
		if e != nil {
			adminErr(w, content.ErrInvalid)
			return
		}
		in := content.Input{Title: q.Title, Body: q.Body, Status: q.Status, Pinned: q.Pinned, TopicID: q.TopicID, SortOrder: q.SortOrder}
		if kind == content.Material {
			in.Body = q.Description
		}
		var x content.Item
		if kind == content.Announcement {
			x, e = a.store.UpdateAnnouncement(r.Context(), id, expected, in)
		} else {
			x, e = a.store.UpdateMaterial(r.Context(), id, expected, in)
		}
		if e != nil {
			adminErr(w, e)
			return
		}
		a.record(r, kind, "updated", x.ID, x.Title)
		writeJSON(w, 200, dataEnvelope{view(x, kind)})
	}
}
func (a adminContent) delete(kind content.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, e := parseID(r)
		if e != nil {
			adminErr(w, e)
			return
		}
		var title string
		if kind == content.Announcement {
			title, e = a.store.DeleteAnnouncement(r.Context(), id)
		} else {
			title, e = a.store.DeleteMaterial(r.Context(), id)
		}
		if e != nil {
			adminErr(w, e)
			return
		}
		a.record(r, kind, "deleted", id, title)
		w.WriteHeader(204)
	}
}
func (a adminContent) record(r *http.Request, k content.Kind, verb string, id int64, title string) {
	if a.audit == nil {
		return
	}
	s, ok := AdminSessionFromContext(r.Context())
	if !ok {
		return
	}
	entity := "announcement"
	if k == content.Material {
		entity = "material"
	}
	uid := s.UserID
	a.audit.Record(r.Context(), AuditEvent{Action: "content." + entity + "_" + verb, Outcome: "success", UserID: &uid, IP: clientIP(r), Detail: map[string]string{"entityId": strconv.FormatInt(id, 10), "title": title}})
}
