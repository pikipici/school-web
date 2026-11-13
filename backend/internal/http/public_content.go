package httpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pikipici/informatika-monitoring/backend/internal/content"
)

const defaultLimit int64 = 20
const maxLimit int64 = 100

type pageMeta struct {
	Page       int64 `json:"page"`
	Limit      int64 `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}
type listEnvelope struct {
	Data any      `json:"data"`
	Meta pageMeta `json:"meta"`
}
type dataEnvelope struct {
	Data any `json:"data"`
}

func parsePage(r *http.Request) (content.Page, pageMeta, error) {
	page, limit := int64(1), defaultLimit
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.ParseInt(raw, 10, 32)
		if err != nil || page < 1 {
			return content.Page{}, pageMeta{}, content.ErrInvalid
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.ParseInt(raw, 10, 32)
		if err != nil || limit < 1 || limit > maxLimit {
			return content.Page{}, pageMeta{}, content.ErrInvalid
		}
	}
	if page-1 > math.MaxInt32/limit {
		return content.Page{}, pageMeta{}, content.ErrInvalid
	}
	return content.Page{Limit: int32(limit), Offset: int32((page - 1) * limit)}, pageMeta{Page: page, Limit: limit}, nil
}
func listMeta(m pageMeta, total int64) pageMeta {
	m.Total = total
	m.TotalPages = (total + m.Limit - 1) / m.Limit
	return m
}
func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		return 0, content.ErrNotFound
	}
	return id, nil
}
func publicError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrNotFound):
		writeError(w, 404, "NOT_FOUND", "Konten tidak ditemukan.")
	case errors.Is(err, content.ErrInvalid):
		writeError(w, 400, "VALIDATION_ERROR", "Parameter permintaan tidak valid.")
	default:
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
	}
}

func listAnnouncements(reader content.PublicReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, m, err := parsePage(r)
		if err != nil {
			publicError(w, err)
			return
		}
		items, total, err := reader.PublicAnnouncements(r.Context(), p)
		if err != nil {
			publicError(w, err)
			return
		}
		writeJSON(w, 200, listEnvelope{items, listMeta(m, total)})
	}
}
func getAnnouncement(reader content.PublicReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			publicError(w, err)
			return
		}
		item, err := reader.PublicAnnouncement(r.Context(), id)
		if err != nil {
			publicError(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{item})
	}
}
func listTopics(reader content.PublicReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := reader.PublicTopics(r.Context())
		if err != nil {
			publicError(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{items})
	}
}
func listMaterials(reader content.PublicReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, m, err := parsePage(r)
		if err != nil {
			publicError(w, err)
			return
		}
		items, total, err := reader.PublicMaterials(r.Context(), p, r.URL.Query().Get("topic"))
		if err != nil {
			publicError(w, err)
			return
		}
		writeJSON(w, 200, listEnvelope{items, listMeta(m, total)})
	}
}
func getMaterial(reader content.PublicReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			publicError(w, err)
			return
		}
		item, err := reader.PublicMaterial(r.Context(), id)
		if err != nil {
			publicError(w, err)
			return
		}
		writeJSON(w, 200, dataEnvelope{item})
	}
}
