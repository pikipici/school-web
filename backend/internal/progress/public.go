package progress

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// Lane values from PLAN.md section 5. They are kept separate on purpose.
type Submission string
type Review string
type Notebook string

const (
	SubmissionNone    Submission = "BELUM"
	SubmissionPartial Submission = "SEBAGIAN"
	SubmissionFull    Submission = "LENGKAP"

	ReviewNotChecked Review = "BELUM_DICEK"

	NotebookNotReceived Notebook = "BELUM_DITERIMA"
	NotebookReceived    Notebook = "SUDAH_DITERIMA"
)

// PublicFilter narrows the public table. Empty fields mean no filter.
// Status is a submission lane (BELUM, SEBAGIAN, LENGKAP) or TUNTAS.
type PublicFilter struct {
	Class  string
	Query  string
	Status string
}

// StatusComplete filters for groups whose three lanes are all done.
const StatusComplete = "TUNTAS"

func (f PublicFilter) keep(g PublicGroup) bool {
	switch f.Status {
	case "":
		return true
	case StatusComplete:
		return g.Complete
	default:
		return string(g.Submission) == f.Status
	}
}

type PublicClass struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type PublicTopic struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

// PublicGroup is one row of the public table. It must never gain an email,
// a note, a finding, or an access-code field.
type PublicGroup struct {
	Name       string      `json:"name"`
	ClassCode  string      `json:"classCode"`
	Topic      PublicTopic `json:"topic"`
	Submission Submission  `json:"submission"`
	Review     Review      `json:"review"`
	Notebook   Notebook    `json:"notebook"`
	Complete   bool        `json:"complete"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type PublicProgress struct {
	Year    string        `json:"year"`
	Classes []PublicClass `json:"classes"`
	Groups  []PublicGroup `json:"groups"`
}

// Summary counts the visible groups by submission lane and completion.
type Summary struct {
	Total    int64 `json:"total"`
	None     int64 `json:"none"`
	Partial  int64 `json:"partial"`
	Full     int64 `json:"full"`
	Complete int64 `json:"complete"`
}

func (f PublicFilter) normalize() (PublicFilter, error) {
	f.Class = strings.TrimSpace(f.Class)
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.ToUpper(strings.TrimSpace(f.Status))
	switch Submission(f.Status) {
	case "", SubmissionNone, SubmissionPartial, SubmissionFull, StatusComplete:
	default:
		return f, fmt.Errorf("%w: status filter", ErrInvalid)
	}
	if f.Class != "" && !validClassCode(f.Class) {
		return f, fmt.Errorf("%w: class filter", ErrInvalid)
	}
	if utf8.RuneCountInString(f.Query) > MaxQuery || !noControl(f.Query) {
		return f, fmt.Errorf("%w: search text", ErrInvalid)
	}
	return f, nil
}

func optText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// PublicProgress returns the active year's active classes and groups.
func (s *Store) PublicProgress(ctx context.Context, f PublicFilter) (PublicProgress, error) {
	f, err := f.normalize()
	if err != nil {
		return PublicProgress{}, err
	}
	out := PublicProgress{Classes: []PublicClass{}, Groups: []PublicGroup{}}
	label, err := s.q.ActiveAcademicYearLabel(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return PublicProgress{}, err
	}
	out.Year = label
	classes, err := s.q.ListPublicClasses(ctx)
	if err != nil {
		return PublicProgress{}, err
	}
	for _, c := range classes {
		out.Classes = append(out.Classes, PublicClass{Code: c.Code, Name: c.Name})
	}
	rows, err := s.q.ListPublicGroups(ctx, dbsql.ListPublicGroupsParams{ClassCode: optText(f.Class), Query: optText(f.Query)})
	if err != nil {
		return PublicProgress{}, err
	}
	for _, r := range rows {
		sub := SubmissionNone
		switch {
		case r.HasPpt && r.HasPdf:
			sub = SubmissionFull
		case r.HasPpt || r.HasPdf:
			sub = SubmissionPartial
		}
		nb := NotebookNotReceived
		if r.NotebookReceived {
			nb = NotebookReceived
		}
		g := PublicGroup{
			Name:       r.Name,
			ClassCode:  r.ClassCode,
			Topic:      PublicTopic{Code: r.TopicCode, Title: r.TopicTitle},
			Submission: sub,
			// Review and completion come with part 3C; until then nothing is
			// approved, so no group can be complete.
			Review:    ReviewNotChecked,
			Notebook:  nb,
			Complete:  false,
			UpdatedAt: r.UpdatedAt.Time,
		}
		if f.keep(g) {
			out.Groups = append(out.Groups, g)
		}
	}
	return out, nil
}

// Summarize counts groups by submission lane and completion, and reports
// the latest update time (nil when there are no groups).
func Summarize(groups []PublicGroup) (Summary, *time.Time) {
	var sum Summary
	var latest *time.Time
	for _, g := range groups {
		sum.Total++
		switch g.Submission {
		case SubmissionNone:
			sum.None++
		case SubmissionPartial:
			sum.Partial++
		case SubmissionFull:
			sum.Full++
		}
		if g.Complete {
			sum.Complete++
		}
		if latest == nil || g.UpdatedAt.After(*latest) {
			t := g.UpdatedAt
			latest = &t
		}
	}
	return sum, latest
}

// PublicSummary counts every group on the unfiltered public table.
func (s *Store) PublicSummary(ctx context.Context) (Summary, error) {
	p, err := s.PublicProgress(ctx, PublicFilter{})
	if err != nil {
		return Summary{}, err
	}
	sum, _ := Summarize(p.Groups)
	return sum, nil
}
