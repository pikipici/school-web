package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

type roundView struct {
	ID          int64     `json:"id"`
	GroupID     int64     `json:"groupId"`
	RoundNumber int32     `json:"roundNumber"`
	ReceivedAt  time.Time `json:"receivedAt"`
	SenderEmail *string   `json:"senderEmail"`
	Subject     string    `json:"subject"`
	PPT         bool      `json:"ppt"`
	PDF         bool      `json:"pdf"`
	PPTFilename string    `json:"pptFilename"`
	PDFFilename string    `json:"pdfFilename"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type notebookView struct {
	Received   bool       `json:"received"`
	ReceivedAt *time.Time `json:"receivedAt"`
	Note       string     `json:"note"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}

func roundOut(r progress.Round) roundView {
	return roundView{r.ID, r.GroupID, r.Number, r.ReceivedAt, r.SenderEmail, r.Subject, r.PPT, r.PDF, r.PPTFile, r.PDFFile, r.Note, r.CreatedAt, r.UpdatedAt}
}

func notebookOut(n progress.NotebookRecord) notebookView {
	return notebookView{n.Received, n.ReceivedAt, n.Note, n.UpdatedAt}
}

type roundRequest struct {
	RoundNumber *int32 `json:"roundNumber"`
	ReceivedAt  string `json:"receivedAt"`
	SenderEmail string `json:"senderEmail"`
	Subject     string `json:"subject"`
	PPT         bool   `json:"ppt"`
	PDF         bool   `json:"pdf"`
	PPTFilename string `json:"pptFilename"`
	PDFFilename string `json:"pdfFilename"`
	Note        string `json:"note"`
	UpdatedAt   string `json:"updatedAt"`
}

type notebookRequest struct {
	Received   bool    `json:"received"`
	ReceivedAt *string `json:"receivedAt"`
	Note       string  `json:"note"`
}

func (q roundRequest) input() (progress.RoundInput, error) {
	at, err := time.Parse(time.RFC3339Nano, q.ReceivedAt)
	if err != nil {
		return progress.RoundInput{}, progress.ErrInvalid
	}
	return progress.RoundInput{ReceivedAt: at, SenderEmail: q.SenderEmail, Subject: q.Subject, PPT: q.PPT, PDF: q.PDF, PPTFile: q.PPTFilename, PDFFile: q.PDFFilename, Note: q.Note}, nil
}

func submissionErr(w http.ResponseWriter, err error) {
	if errors.Is(err, progress.ErrNotLast) {
		writeError(w, 409, "NOT_LAST_ROUND", "Hanya putaran terakhir yang bisa dihapus.")
		return
	}
	progressErr(w, err)
}

func actorOf(r *http.Request) progress.Actor {
	a := progress.Actor{IP: clientIP(r)}
	if s, ok := AdminSessionFromContext(r.Context()); ok {
		a.UserID = s.UserID
	}
	return a
}

func (a adminProgress) listRounds(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	rs, err := a.store.Rounds(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	out := make([]roundView, 0, len(rs))
	for _, x := range rs {
		out = append(out, roundOut(x))
	}
	writeJSON(w, 200, dataEnvelope{map[string]any{"submission": progress.SubmissionOf(rs), "rounds": out}})
}

func (a adminProgress) recordRound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	q, ok := decodeJSON[roundRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	in, err := q.input()
	if err != nil || q.RoundNumber == nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	x, err := a.store.RecordRound(r.Context(), id, *q.RoundNumber, in, actorOf(r))
	if err != nil {
		submissionErr(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/admin/submissions/"+strconv.FormatInt(x.ID, 10))
	writeJSON(w, 201, dataEnvelope{roundOut(x)})
}

func (a adminProgress) updateRound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	q, ok := decodeJSON[roundRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	in, err := q.input()
	expected, serr := parseStamp(q.UpdatedAt)
	if err != nil || serr != nil {
		progressErr(w, progress.ErrInvalid)
		return
	}
	x, err := a.store.UpdateRound(r.Context(), id, expected, in, actorOf(r))
	if err != nil {
		submissionErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{roundOut(x)})
}

func (a adminProgress) deleteRound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	if _, err := a.store.DeleteRound(r.Context(), id, actorOf(r)); err != nil {
		submissionErr(w, err)
		return
	}
	w.WriteHeader(204)
}

func (a adminProgress) getNotebook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	n, err := a.store.Notebook(r.Context(), id)
	if err != nil {
		progressErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{notebookOut(n)})
}

func (a adminProgress) putNotebook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		progressErr(w, progress.ErrNotFound)
		return
	}
	q, ok := decodeJSON[notebookRequest](w, r, maxProgressBody)
	if !ok {
		return
	}
	in := progress.NotebookInput{Received: q.Received, Note: q.Note}
	if q.ReceivedAt != nil && *q.ReceivedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, *q.ReceivedAt)
		if err != nil {
			progressErr(w, progress.ErrInvalid)
			return
		}
		in.ReceivedAt = &t
	}
	n, err := a.store.SetNotebook(r.Context(), id, in, actorOf(r))
	if err != nil {
		progressErr(w, err)
		return
	}
	writeJSON(w, 200, dataEnvelope{notebookOut(n)})
}

// submissionRoutes mounts the 3B admin API. Audit is written by the store in
// the same transaction as each change, not by the handler.
func (a adminProgress) submissionRoutes(r chi.Router) {
	r.Get("/groups/{id}/submissions", a.listRounds)
	r.Post("/groups/{id}/submissions", a.recordRound)
	r.Patch("/submissions/{id}", a.updateRound)
	r.Delete("/submissions/{id}", a.deleteRound)
	r.Get("/groups/{id}/notebook", a.getNotebook)
	r.Put("/groups/{id}/notebook", a.putNotebook)
}
