package httpapi

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/attachments"
)

// downloadDeadline replaces the server-wide 15 s write timeout for a
// download: 25 MB to a slow school connection can take minutes.
const downloadDeadline = 10 * time.Minute

type publicAttachments struct {
	svc *attachments.Service
}

// publicAttachmentView is all a visitor sees: no storage key, hash, or dates.
type publicAttachmentView struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

func (p publicAttachments) list(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "Konten tidak ditemukan.")
		return
	}
	xs, err := p.svc.PublicList(r.Context(), id)
	if errors.Is(err, attachments.ErrMaterialNotFound) {
		writeError(w, 404, "NOT_FOUND", "Konten tidak ditemukan.")
		return
	}
	if err != nil {
		slog.Error("public attachment list failed", "error", err)
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
		return
	}
	out := make([]publicAttachmentView, 0, len(xs))
	for _, x := range xs {
		out = append(out, publicAttachmentView{ID: x.ID, Name: x.Name, Kind: x.Kind.String(), Size: x.Size})
	}
	writeJSON(w, 200, dataEnvelope{out})
}

// download streams the file as an attachment. The type comes from content
// validation at upload, never from the name; the browser is told not to
// sniff it, not to render it inline, and to sandbox it if it ever does.
// http.ServeContent handles Range, If-Range, and If-None-Match.
func (p publicAttachments) download(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "Lampiran tidak ditemukan.")
		return
	}
	a, f, err := p.svc.PublicOpen(r.Context(), id)
	if errors.Is(err, attachments.ErrNotFound) {
		writeError(w, 404, "NOT_FOUND", "Lampiran tidak ditemukan.")
		return
	}
	if err != nil {
		slog.Error("public attachment open failed", "error", err)
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
		return
	}
	defer f.Close()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(downloadDeadline))

	h := w.Header()
	h.Set("Content-Type", a.Kind.ContentType())
	h.Set("Content-Disposition", contentDisposition(a.Name))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	// Revalidate every time so unpublishing a material takes effect at once.
	h.Set("Cache-Control", "no-cache")
	h.Set("ETag", `"`+hex.EncodeToString(a.SHA256)+`"`)
	http.ServeContent(w, r, "", a.CreatedAt, f)
}

// contentDisposition builds an attachment header with an ASCII fallback name
// and the exact UTF-8 name per RFC 6266 / RFC 5987.
func contentDisposition(name string) string {
	var fallback, encoded strings.Builder
	for _, c := range name {
		if c >= 0x20 && c < 0x7f && c != '"' && c != '\\' {
			fallback.WriteRune(c)
		} else {
			fallback.WriteByte('_')
		}
	}
	for _, b := range []byte(name) {
		if isAttrChar(b) {
			encoded.WriteByte(b)
		} else {
			fmt.Fprintf(&encoded, "%%%02X", b)
		}
	}
	return `attachment; filename="` + fallback.String() + `"; filename*=UTF-8''` + encoded.String()
}

func isAttrChar(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	}
	return strings.IndexByte("!#$&+-.^_`|~", b) >= 0
}
