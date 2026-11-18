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
	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

const (
	ReviewChecking         Review = "DIPERIKSA"
	ReviewNeedsRevision    Review = "PERLU_REVISI"
	ReviewRevisionReceived Review = "REVISI_DITERIMA"
	ReviewApproved         Review = "DISETUJUI"
)

var (
	ErrPublished  = errors.New("progress: published finding cannot be deleted")
	ErrReviewRule = errors.New("progress: review requirements not met")
)

type FindingInput struct{ FileType, Location, Problem, Action, Priority string }
type Finding struct {
	ID          int64      `json:"id"`
	GroupID     int64      `json:"groupId"`
	FileType    string     `json:"fileType"`
	Location    string     `json:"location"`
	Problem     string     `json:"problem"`
	Action      string     `json:"action"`
	Priority    string     `json:"priority"`
	Resolved    bool       `json:"resolved"`
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
type ReviewRecord struct {
	Status    Review    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type PublicFinding struct {
	ID          int64     `json:"id"`
	FileType    string    `json:"fileType"`
	Location    string    `json:"location"`
	Problem     string    `json:"problem"`
	Action      string    `json:"action"`
	Priority    string    `json:"priority"`
	Resolved    bool      `json:"resolved"`
	PublishedAt time.Time `json:"publishedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type PublicDetail struct {
	PublicGroup
	Findings []PublicFinding `json:"findings"`
}

func ValidateFinding(in FindingInput) (FindingInput, error) {
	in.FileType = strings.TrimSpace(in.FileType)
	in.Location = strings.TrimSpace(in.Location)
	in.Problem = strings.TrimSpace(in.Problem)
	in.Action = strings.TrimSpace(in.Action)
	in.Priority = strings.TrimSpace(in.Priority)
	switch in.FileType {
	case "PPT", "PDF", "BUKU", "PRESENTASI":
	default:
		return in, ErrInvalid
	}
	switch in.Priority {
	case "WAJIB", "SARAN":
	default:
		return in, ErrInvalid
	}
	for _, v := range []struct {
		s   string
		max int
	}{{in.Location, 200}, {in.Problem, 2000}, {in.Action, 2000}} {
		if v.s == "" || utf8.RuneCountInString(v.s) > v.max || !noControl(v.s) {
			return in, ErrInvalid
		}
	}
	return in, nil
}
func findingOut(v dbsql.EvaluationFinding) Finding {
	f := Finding{ID: v.ID, GroupID: v.GroupID, FileType: v.FileType, Location: v.Location, Problem: v.Problem, Action: v.RequiredAction, Priority: v.Priority, Resolved: v.Resolved, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
	if v.PublishedAt.Valid {
		at := v.PublishedAt.Time
		f.PublishedAt = &at
	}
	return f
}
func (s *Store) Findings(ctx context.Context, id int64) ([]Finding, error) {
	if _, err := s.Group(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.q.ListFindings(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Finding, 0, len(rows))
	for _, r := range rows {
		out = append(out, findingOut(r))
	}
	return out, nil
}
func (s *Store) CreateFinding(ctx context.Context, id int64, in FindingInput, a Actor) (Finding, error) {
	in, err := ValidateFinding(in)
	if err != nil {
		return Finding{}, err
	}
	var out Finding
	err = s.change(ctx, id, a, "finding_created", func() map[string]string { return map[string]string{"findingId": strconv.FormatInt(out.ID, 10)} }, func(q *dbsql.Queries, now time.Time) error {
		v, e := q.InsertFinding(ctx, dbsql.InsertFindingParams{GroupID: id, FileType: in.FileType, Location: in.Location, Problem: in.Problem, RequiredAction: in.Action, Priority: in.Priority, CreatedBy: actorID(a), CreatedAt: stamp(now)})
		out = findingOut(v)
		return e
	})
	return out, err
}
func (s *Store) UpdateFinding(ctx context.Context, id int64, expected time.Time, in FindingInput, resolved bool, a Actor) (Finding, error) {
	in, err := ValidateFinding(in)
	if err != nil {
		return Finding{}, err
	}
	old, err := s.q.GetFinding(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Finding{}, ErrNotFound
	}
	if err != nil {
		return Finding{}, err
	}
	var out Finding
	err = s.change(ctx, old.GroupID, a, "finding_updated", func() map[string]string { return map[string]string{"findingId": strconv.FormatInt(id, 10)} }, func(q *dbsql.Queries, now time.Time) error {
		current, e := q.GetFinding(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if current.PublishedAt.Valid && (current.FileType != in.FileType || current.Location != in.Location || current.Problem != in.Problem || current.RequiredAction != in.Action || current.Priority != in.Priority) {
			return ErrPublished
		}
		v, e := q.UpdateFinding(ctx, dbsql.UpdateFindingParams{ID: id, UpdatedAt: stamp(expected), FileType: in.FileType, Location: in.Location, Problem: in.Problem, RequiredAction: in.Action, Priority: in.Priority, Resolved: resolved, UpdatedAt_2: stamp(now)})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrConflict
		}
		out = findingOut(v)
		return e
	})
	return out, err
}
func (s *Store) DeleteFinding(ctx context.Context, id int64, expected time.Time, a Actor) error {
	old, err := s.q.GetFinding(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.change(ctx, old.GroupID, a, "finding_deleted", func() map[string]string { return map[string]string{"findingId": strconv.FormatInt(id, 10)} }, func(q *dbsql.Queries, now time.Time) error {
		v, e := q.GetFinding(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if v.PublishedAt.Valid {
			return ErrPublished
		}
		n, e := q.DeleteDraftFinding(ctx, dbsql.DeleteDraftFindingParams{ID: id, UpdatedAt: stamp(expected)})
		if e == nil && n == 0 {
			return ErrConflict
		}
		return e
	})
}

// PublishReview commits publication, the intended review state, and audit together.
func (s *Store) PublishReview(ctx context.Context, id int64, expected time.Time, status Review, a Actor) error {
	switch status {
	case ReviewNotChecked, ReviewChecking, ReviewNeedsRevision, ReviewRevisionReceived, ReviewApproved:
	default:
		return ErrInvalid
	}
	return s.change(ctx, id, a, "findings_published", func() map[string]string { return map[string]string{"status": string(status)} }, func(q *dbsql.Queries, now time.Time) error {
		if _, err := q.SetReview(ctx, dbsql.SetReviewParams{ID: id, ReviewUpdatedAt: stamp(expected), ReviewStatus: string(status), ReviewUpdatedAt_2: stamp(now)}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrConflict
			}
			return err
		}
		n, err := q.PublishFindings(ctx, dbsql.PublishFindingsParams{GroupID: id, PublishedAt: stamp(now)})
		if err == nil && n == 0 {
			return ErrConflict
		}
		return err
	})
}
func (s *Store) GetReview(ctx context.Context, id int64) (ReviewRecord, error) {
	v, err := s.q.GetReview(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewRecord{}, ErrNotFound
	}
	return ReviewRecord{Review(v.ReviewStatus), v.ReviewUpdatedAt.Time}, err
}
func (s *Store) SetReview(ctx context.Context, id int64, expected time.Time, status Review, a Actor) (ReviewRecord, error) {
	switch status {
	case ReviewNotChecked, ReviewChecking, ReviewNeedsRevision, ReviewRevisionReceived, ReviewApproved:
	default:
		return ReviewRecord{}, ErrInvalid
	}
	var out ReviewRecord
	err := s.change(ctx, id, a, "review_updated", func() map[string]string { return map[string]string{"status": string(status)} }, func(q *dbsql.Queries, now time.Time) error {
		v, err := q.SetReview(ctx, dbsql.SetReviewParams{ID: id, ReviewUpdatedAt: stamp(expected), ReviewStatus: string(status), ReviewUpdatedAt_2: stamp(now)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		out = ReviewRecord{Review(v.ReviewStatus), v.ReviewUpdatedAt.Time}
		return err
	})
	return out, err
}
func (s *Store) PublicDetail(ctx context.Context, id int64) (PublicDetail, error) {
	r, err := s.q.GetPublicGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicDetail{}, ErrNotFound
	}
	if err != nil {
		return PublicDetail{}, err
	}
	g := publicGroup(r.ID, r.Name, r.ClassCode, r.TopicCode, r.TopicTitle, r.ReviewStatus, r.HasPpt, r.HasPdf, r.NotebookReceived, r.UpdatedAt.Time)
	rows, err := s.q.ListPublicFindings(ctx, id)
	if err != nil {
		return PublicDetail{}, fmt.Errorf("progress: findings: %w", err)
	}
	out := PublicDetail{PublicGroup: g, Findings: []PublicFinding{}}
	for _, v := range rows {
		out.Findings = append(out.Findings, PublicFinding{v.ID, v.FileType, v.Location, v.Problem, v.RequiredAction, v.Priority, v.Resolved, v.PublishedAt.Time, v.UpdatedAt.Time})
	}
	return out, nil
}
func publicGroup(id int64, name, class, code, title, review string, ppt, pdf, notebook bool, at time.Time) PublicGroup {
	sub := SubmissionNone
	if ppt && pdf {
		sub = SubmissionFull
	} else if ppt || pdf {
		sub = SubmissionPartial
	}
	nb := NotebookNotReceived
	if notebook {
		nb = NotebookReceived
	}
	return PublicGroup{ID: id, Name: name, ClassCode: class, Topic: PublicTopic{code, title}, Submission: sub, Review: Review(review), Notebook: nb, Complete: sub == SubmissionFull && Review(review) == ReviewApproved && notebook, UpdatedAt: at}
}
