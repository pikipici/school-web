package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/attachments"
)

const (
	// multipartOverhead allows for boundaries and part headers around the file.
	multipartOverhead = 64 << 10
	// uploadDeadline replaces the server-wide 15 s read/write timeouts for an
	// upload: 25 MB over a slow school connection can take minutes.
	uploadDeadline = 10 * time.Minute
)

type adminAttachments struct {
	svc   *attachments.Service
	audit AuditRecorder
}

type attachmentView struct {
	ID          int64     `json:"id"`
	MaterialID  int64     `json:"materialId"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}

func attachmentOut(a attachments.Attachment) attachmentView {
	return attachmentView{ID: a.ID, MaterialID: a.MaterialID, Name: a.Name, Kind: a.Kind.String(), ContentType: a.Kind.ContentType(), Size: a.Size, CreatedAt: a.CreatedAt}
}

func attachmentErr(w http.ResponseWriter, err error) {
	var big *http.MaxBytesError
	switch {
	case errors.Is(err, attachments.ErrMaterialNotFound):
		writeError(w, 404, "NOT_FOUND", "Materi tidak ditemukan.")
	case errors.Is(err, attachments.ErrNotFound):
		writeError(w, 404, "NOT_FOUND", "Lampiran tidak ditemukan.")
	case errors.Is(err, attachments.ErrLimit):
		writeError(w, 409, "ATTACHMENT_LIMIT", "Materi ini sudah punya 5 lampiran.")
	case errors.Is(err, attachments.ErrBusy):
		w.Header().Set("Retry-After", "5")
		writeError(w, 503, "SERVICE_BUSY", "Unggahan lain sedang berjalan. Coba lagi sebentar.")
	case errors.Is(err, attachments.ErrTooLarge), errors.As(err, &big):
		writeError(w, 413, "PAYLOAD_TOO_LARGE", "Berkas lebih dari 25 MB.")
	case errors.Is(err, attachments.ErrMacro):
		writeError(w, 415, "MACRO_NOT_ALLOWED", "Presentasi bermakro tidak diterima.")
	case errors.Is(err, attachments.ErrType):
		writeError(w, 415, "UNSUPPORTED_FILE_TYPE", "Hanya berkas PDF, PPT, atau PPTX yang diterima.")
	case errors.Is(err, attachments.ErrQuota):
		writeError(w, 507, "STORAGE_QUOTA_EXCEEDED", "Kuota penyimpanan lampiran sudah penuh.")
	case errors.Is(err, attachments.ErrDiskFull):
		writeError(w, 507, "INSUFFICIENT_STORAGE", "Ruang disk server tidak cukup.")
	case errors.Is(err, attachments.ErrName):
		writeError(w, 400, "INVALID_FILE_NAME", "Nama berkas tidak dapat dipakai.")
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, multipart.ErrMessageTooLarge):
		writeError(w, 400, "INVALID_REQUEST", "Unggahan terputus atau tidak valid.")
	default:
		slog.Error("attachment request failed", "error", err)
		writeError(w, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
	}
}

// upload accepts multipart/form-data with one part named "file". The file is
// streamed to disk by the attachments store and never held in memory.
func (a adminAttachments) upload(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "Materi tidak ditemukan.")
		return
	}
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Gunakan Content-Type multipart/form-data.")
		return
	}
	rc := http.NewResponseController(w)
	deadline := time.Now().Add(uploadDeadline)
	_ = rc.SetReadDeadline(deadline) // unsupported only in tests' recorders
	_ = rc.SetWriteDeadline(deadline)

	r.Body = http.MaxBytesReader(w, r.Body, a.svc.MaxFile()+multipartOverhead)
	part, err := multipart.NewReader(r.Body, params["boundary"]).NextPart()
	if err != nil {
		var big *http.MaxBytesError
		if errors.As(err, &big) {
			attachmentErr(w, err)
			return
		}
		writeError(w, 400, "INVALID_REQUEST", "Unggahan tidak valid.")
		return
	}
	defer part.Close()
	if part.FormName() != "file" || part.FileName() == "" {
		writeError(w, 400, "INVALID_REQUEST", "Kirim satu berkas pada isian \"file\".")
		return
	}
	x, err := a.svc.Upload(r.Context(), id, part, part.FileName())
	if err != nil {
		attachmentErr(w, err)
		return
	}
	a.record(r, "uploaded", x)
	w.Header().Set("Location", "/api/v1/admin/attachments/"+strconv.FormatInt(x.ID, 10))
	writeJSON(w, 201, dataEnvelope{attachmentOut(x)})
}

func (a adminAttachments) list(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "Materi tidak ditemukan.")
		return
	}
	xs, err := a.svc.List(r.Context(), id)
	if err != nil {
		attachmentErr(w, err)
		return
	}
	out := make([]attachmentView, 0, len(xs))
	for _, x := range xs {
		out = append(out, attachmentOut(x))
	}
	writeJSON(w, 200, dataEnvelope{out})
}

func (a adminAttachments) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "Lampiran tidak ditemukan.")
		return
	}
	x, err := a.svc.Delete(r.Context(), id)
	if err != nil {
		attachmentErr(w, err)
		return
	}
	a.record(r, "deleted", x)
	w.WriteHeader(204)
}

func (a adminAttachments) record(r *http.Request, verb string, x attachments.Attachment) {
	if a.audit == nil {
		return
	}
	s, ok := AdminSessionFromContext(r.Context())
	if !ok {
		return
	}
	uid := s.UserID
	a.audit.Record(r.Context(), AuditEvent{
		Action:  "content.attachment_" + verb,
		Outcome: "success",
		UserID:  &uid,
		IP:      clientIP(r),
		Detail: map[string]string{
			"entityId":   strconv.FormatInt(x.ID, 10),
			"materialId": strconv.FormatInt(x.MaterialID, 10),
			"title":      x.Name,
		},
	})
}
