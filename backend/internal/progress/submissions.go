package progress

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// ErrNotLast means only the newest round of a group may be deleted.
var ErrNotLast = errors.New("progress: only the last round can be deleted")

// Limits, mirrored by migration 00009.
const (
	MaxSubject      = 300
	MaxFileName     = 200
	MaxRoundNote    = 2000
	MaxNotebookNote = 1000
	// clockSkew is how far in the future a received time may be.
	clockSkew = 5 * time.Minute
)

// Actor is who made a change, for the audit trail.
type Actor struct {
	UserID int64
	IP     string
}

// RoundInput is what the teacher types after checking Gmail by hand.
type RoundInput struct {
	ReceivedAt  time.Time
	SenderEmail string
	Subject     string
	PPT, PDF    bool
	PPTFile     string
	PDFFile     string
	Note        string
}

type Round struct {
	ID          int64
	GroupID     int64
	Number      int32
	ReceivedAt  time.Time
	SenderEmail *string
	Subject     string
	PPT, PDF    bool
	PPTFile     string
	PDFFile     string
	Note        string
	RecordedBy  *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NotebookInput struct {
	Received   bool
	ReceivedAt *time.Time // nil with Received means "now"
	Note       string
}

type NotebookRecord struct {
	GroupID    int64
	Received   bool
	ReceivedAt *time.Time
	Note       string
	UpdatedAt  *time.Time // nil until first saved
}

func validFileName(s string) bool {
	return utf8.RuneCountInString(s) <= MaxFileName && noControl(s) && !strings.ContainsAny(s, `/\`)
}

// ValidateRound normalises in and checks it against the same rules as the
// database. now bounds how far in the future ReceivedAt may be.
func ValidateRound(in RoundInput, now time.Time) (RoundInput, error) {
	in.SenderEmail = strings.ToLower(strings.TrimSpace(in.SenderEmail))
	in.Subject = strings.TrimSpace(in.Subject)
	in.PPTFile = strings.TrimSpace(in.PPTFile)
	in.PDFFile = strings.TrimSpace(in.PDFFile)
	in.Note = strings.TrimSpace(in.Note)
	switch {
	case in.ReceivedAt.IsZero():
		return RoundInput{}, fmt.Errorf("%w: received time is required", ErrInvalid)
	case in.ReceivedAt.After(now.Add(clockSkew)):
		return RoundInput{}, fmt.Errorf("%w: received time is in the future", ErrInvalid)
	case !in.PPT && !in.PDF:
		return RoundInput{}, fmt.Errorf("%w: tick PPT, PDF, or both", ErrInvalid)
	case (!in.PPT && in.PPTFile != "") || (!in.PDF && in.PDFFile != ""):
		return RoundInput{}, fmt.Errorf("%w: file name given for a file that was not received", ErrInvalid)
	case !validFileName(in.PPTFile) || !validFileName(in.PDFFile):
		return RoundInput{}, fmt.Errorf("%w: file name", ErrInvalid)
	case in.SenderEmail != "" && (len(in.SenderEmail) > MaxEmail || !emailPattern.MatchString(in.SenderEmail)):
		return RoundInput{}, fmt.Errorf("%w: sender email", ErrInvalid)
	case utf8.RuneCountInString(in.Subject) > MaxSubject || !noControl(in.Subject):
		return RoundInput{}, fmt.Errorf("%w: subject", ErrInvalid)
	case utf8.RuneCountInString(in.Note) > MaxRoundNote:
		return RoundInput{}, fmt.Errorf("%w: note too long", ErrInvalid)
	}
	return in, nil
}

// SubmissionOf computes the submission lane from all rounds of a group.
func SubmissionOf(rounds []Round) Submission {
	var ppt, pdf bool
	for _, r := range rounds {
		ppt = ppt || r.PPT
		pdf = pdf || r.PDF
	}
	switch {
	case ppt && pdf:
		return SubmissionFull
	case ppt || pdf:
		return SubmissionPartial
	}
	return SubmissionNone
}

func roundOut(v dbsql.SubmissionRound) Round {
	r := Round{ID: v.ID, GroupID: v.GroupID, Number: v.RoundNumber, ReceivedAt: v.ReceivedAt.Time, Subject: v.EmailSubject, PPT: v.PptReceived, PDF: v.PdfReceived, PPTFile: v.PptFilename, PDFFile: v.PdfFilename, Note: v.InternalNote, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
	if v.SenderEmail.Valid {
		e := v.SenderEmail.String
		r.SenderEmail = &e
	}
	if v.RecordedBy.Valid {
		id := v.RecordedBy.Int64
		r.RecordedBy = &id
	}
	return r
}

func notebookOut(v dbsql.GroupNotebook) NotebookRecord {
	n := NotebookRecord{GroupID: v.GroupID, Received: v.IsReceived, Note: v.Note}
	if v.ReceivedAt.Valid {
		t := v.ReceivedAt.Time
		n.ReceivedAt = &t
	}
	u := v.UpdatedAt.Time
	n.UpdatedAt = &u
	return n
}

// change runs fn in a transaction with the group row locked, then bumps the
// group's progress time and writes the audit row in the same transaction. If
// any step fails, nothing is kept.
func (s *Store) change(ctx context.Context, groupID int64, a Actor, action string, detail func() map[string]string, fn func(q *dbsql.Queries, now time.Time) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	q := s.q.WithTx(tx)
	g, err := q.LockGroup(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	now := s.now()
	if err := fn(q, now); err != nil {
		return err
	}
	valid, err := q.ReviewStateValid(ctx, groupID)
	if err != nil {
		return err
	}
	if !valid {
		return ErrReviewRule
	}
	if err := q.TouchGroupProgress(ctx, dbsql.TouchGroupProgressParams{ID: groupID, ProgressUpdatedAt: stamp(now)}); err != nil {
		return err
	}
	d := map[string]string{"entityId": strconv.FormatInt(groupID, 10), "title": g.Name}
	for k, v := range detail() {
		d[k] = v
	}
	uid := a.UserID
	if err := audit.Write(ctx, q, now, audit.Event{Action: "progress." + action, Outcome: "success", UserID: &uid, IP: a.IP, Detail: d}); err != nil {
		return fmt.Errorf("progress: audit: %w", err)
	}
	return tx.Commit(ctx)
}

func actorID(a Actor) pgtype.Int8 { return pgtype.Int8{Int64: a.UserID, Valid: a.UserID > 0} }

func roundWriteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch {
		case pg.Code == "23505":
			// Same round number twice: a double click or a stale form.
			return ErrConflict
		case pg.Code == "23514" && pg.ConstraintName == "":
			// Raised by submission_rounds_next_number(): not the next round.
			return ErrConflict
		}
	}
	return writeError(err)
}

// Rounds lists a group's rounds, oldest first.
func (s *Store) Rounds(ctx context.Context, groupID int64) ([]Round, error) {
	ok, err := s.q.GroupExists(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListRounds(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]Round, 0, len(rows))
	for _, v := range rows {
		out = append(out, roundOut(v))
	}
	return out, nil
}

// RecordRound adds round number to a group. number must be the next one
// (0 for the first); sending the same number twice gives ErrConflict, so a
// double click cannot create two rounds.
func (s *Store) RecordRound(ctx context.Context, groupID int64, number int32, in RoundInput, a Actor) (Round, error) {
	in, err := ValidateRound(in, s.now())
	if err != nil {
		return Round{}, err
	}
	if number < 0 {
		return Round{}, fmt.Errorf("%w: round number", ErrInvalid)
	}
	var out Round
	err = s.change(ctx, groupID, a, "round_recorded", func() map[string]string {
		return map[string]string{"round": strconv.Itoa(int(number))}
	}, func(q *dbsql.Queries, now time.Time) error {
		v, err := q.InsertRound(ctx, dbsql.InsertRoundParams{
			GroupID: groupID, RoundNumber: number, ReceivedAt: stamp(in.ReceivedAt), SenderEmail: email(in.SenderEmail),
			EmailSubject: in.Subject, PptReceived: in.PPT, PdfReceived: in.PDF, PptFilename: in.PPTFile, PdfFilename: in.PDFFile,
			InternalNote: in.Note, RecordedBy: actorID(a), CreatedAt: stamp(now),
		})
		if err != nil {
			return roundWriteError(err)
		}
		out = roundOut(v)
		return nil
	})
	return out, err
}

// UpdateRound corrects a round if it still has updatedAt == expected.
func (s *Store) UpdateRound(ctx context.Context, id int64, expected time.Time, in RoundInput, a Actor) (Round, error) {
	in, err := ValidateRound(in, s.now())
	if err != nil {
		return Round{}, err
	}
	cur, err := s.q.GetRound(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Round{}, ErrNotFound
	}
	if err != nil {
		return Round{}, err
	}
	var out Round
	err = s.change(ctx, cur.GroupID, a, "round_updated", func() map[string]string {
		return map[string]string{"round": strconv.Itoa(int(cur.RoundNumber))}
	}, func(q *dbsql.Queries, now time.Time) error {
		v, err := q.UpdateRound(ctx, dbsql.UpdateRoundParams{
			ID: id, UpdatedAt: stamp(expected), ReceivedAt: stamp(in.ReceivedAt), SenderEmail: email(in.SenderEmail),
			EmailSubject: in.Subject, PptReceived: in.PPT, PdfReceived: in.PDF, PptFilename: in.PPTFile, PdfFilename: in.PDFFile,
			InternalNote: in.Note, RecordedBy: actorID(a), UpdatedAt_2: stamp(now),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			if _, gerr := q.GetRound(ctx, id); errors.Is(gerr, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return ErrConflict
		}
		if err != nil {
			return writeError(err)
		}
		out = roundOut(v)
		return nil
	})
	return out, err
}

// DeleteRound removes the newest round of its group (a typo fix). Older
// rounds stay: deleting them would renumber history.
func (s *Store) DeleteRound(ctx context.Context, id int64, a Actor) (Round, error) {
	cur, err := s.q.GetRound(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Round{}, ErrNotFound
	}
	if err != nil {
		return Round{}, err
	}
	out := roundOut(cur)
	err = s.change(ctx, cur.GroupID, a, "round_deleted", func() map[string]string {
		return map[string]string{"round": strconv.Itoa(int(cur.RoundNumber))}
	}, func(q *dbsql.Queries, _ time.Time) error {
		last, err := q.LastRoundNumber(ctx, cur.GroupID)
		if err != nil {
			return err
		}
		if last != cur.RoundNumber {
			// Also covers a round deleted by someone else meanwhile.
			if _, gerr := q.GetRound(ctx, id); errors.Is(gerr, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return ErrNotLast
		}
		return deleteError(q.DeleteRound(ctx, id))
	})
	return out, err
}

// Notebook returns a group's notebook record; a group with none yet is
// reported as not received.
func (s *Store) Notebook(ctx context.Context, groupID int64) (NotebookRecord, error) {
	v, err := s.q.GetNotebook(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		ok, gerr := s.q.GroupExists(ctx, groupID)
		if gerr != nil {
			return NotebookRecord{}, gerr
		}
		if !ok {
			return NotebookRecord{}, ErrNotFound
		}
		return NotebookRecord{GroupID: groupID}, nil
	}
	if err != nil {
		return NotebookRecord{}, err
	}
	return notebookOut(v), nil
}

// SetNotebook records whether the group's notebook was handed in.
func (s *Store) SetNotebook(ctx context.Context, groupID int64, in NotebookInput, a Actor) (NotebookRecord, error) {
	in.Note = strings.TrimSpace(in.Note)
	now := s.now()
	if utf8.RuneCountInString(in.Note) > MaxNotebookNote {
		return NotebookRecord{}, fmt.Errorf("%w: note too long", ErrInvalid)
	}
	var at pgtype.Timestamptz
	if in.Received {
		t := now
		if in.ReceivedAt != nil {
			t = *in.ReceivedAt
		}
		if t.After(now.Add(clockSkew)) {
			return NotebookRecord{}, fmt.Errorf("%w: received time is in the future", ErrInvalid)
		}
		at = stamp(t)
	}
	var out NotebookRecord
	err := s.change(ctx, groupID, a, "notebook_updated", func() map[string]string {
		return map[string]string{"received": strconv.FormatBool(in.Received)}
	}, func(q *dbsql.Queries, now time.Time) error {
		v, err := q.UpsertNotebook(ctx, dbsql.UpsertNotebookParams{GroupID: groupID, IsReceived: in.Received, ReceivedAt: at, Note: in.Note, UpdatedBy: actorID(a), CreatedAt: stamp(now)})
		if err != nil {
			return writeError(err)
		}
		out = notebookOut(v)
		return nil
	})
	return out, err
}
