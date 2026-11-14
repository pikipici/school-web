// Package progress manages academic years, classes, and study groups for the
// Progres Tugas Kelompok module, and builds the public progress table.
//
// The public view never carries a group's email or any internal note. In
// part 3A every group is shown as not yet submitted; later parts compute the
// three lanes from the teacher's records.
package progress

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

var (
	ErrInvalid          = errors.New("progress: invalid input")
	ErrInvalidReference = errors.New("progress: referenced year, class, or topic does not exist")
	ErrDuplicate        = errors.New("progress: duplicate name")
	ErrConflict         = errors.New("progress: changed by someone else")
	ErrNotFound         = errors.New("progress: not found")
	ErrInUse            = errors.New("progress: still has records; deactivate instead")
)

// Limits, mirrored by migration 00008.
const (
	MaxClassCode = 20
	MaxClassName = 80
	MaxGroupName = 60
	MaxEmail     = 254
	MaxQuery     = 60
)

var (
	yearPattern  = regexp.MustCompile(`^([0-9]{4})/([0-9]{4})$`)
	classPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z .-]*$`)
	emailPattern = regexp.MustCompile(`^[a-z0-9._%+-]+@[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$`)
)

type Year struct {
	ID        int64
	Label     string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Class struct {
	ID         int64
	YearID     int64
	Code       string
	Name       string
	Active     bool
	GroupCount int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ClassInput is a class to create or update. Active is ignored on create:
// new classes always start active so they appear on the public table.
type ClassInput struct {
	YearID int64
	Code   string
	Name   string
	Active bool
}

type Group struct {
	ID        int64
	ClassID   int64
	TopicID   int64
	Name      string
	Email     *string
	SortOrder int32
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GroupInput is a group to create or update. An empty Email means none.
// Active is ignored on create.
type GroupInput struct {
	ClassID   int64
	TopicID   int64
	Name      string
	Email     string
	SortOrder int32
	Active    bool
}

func noControl(s string) bool {
	return !strings.ContainsFunc(s, unicode.IsControl)
}

// ValidateYearLabel accepts "YYYY/YYYY" where the second year follows the first.
func ValidateYearLabel(label string) (string, error) {
	label = strings.TrimSpace(label)
	m := yearPattern.FindStringSubmatch(label)
	if m == nil {
		return "", fmt.Errorf("%w: academic year must look like 2026/2027", ErrInvalid)
	}
	var a, b int
	fmt.Sscan(m[1], &a)
	fmt.Sscan(m[2], &b)
	if b != a+1 {
		return "", fmt.Errorf("%w: academic year must span two consecutive years", ErrInvalid)
	}
	return label, nil
}

func validClassCode(code string) bool {
	n := utf8.RuneCountInString(code)
	return n >= 1 && n <= MaxClassCode && classPattern.MatchString(code)
}

func ValidateClass(in ClassInput) (ClassInput, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if in.YearID < 1 {
		return ClassInput{}, fmt.Errorf("%w: academic year is required", ErrInvalid)
	}
	if !validClassCode(in.Code) {
		return ClassInput{}, fmt.Errorf("%w: class code must be 1-20 letters, digits, spaces, dots, or dashes", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Name) > MaxClassName || !noControl(in.Name) {
		return ClassInput{}, fmt.Errorf("%w: class name too long or has control characters", ErrInvalid)
	}
	return in, nil
}

func ValidateGroup(in GroupInput) (GroupInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.ClassID < 1 || in.TopicID < 1 {
		return GroupInput{}, fmt.Errorf("%w: class and topic are required", ErrInvalid)
	}
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > MaxGroupName || !noControl(in.Name) {
		return GroupInput{}, fmt.Errorf("%w: group name must be 1-60 characters", ErrInvalid)
	}
	if in.Email != "" && (len(in.Email) > MaxEmail || !emailPattern.MatchString(in.Email)) {
		return GroupInput{}, fmt.Errorf("%w: group email is not valid", ErrInvalid)
	}
	if in.SortOrder < 0 {
		return GroupInput{}, fmt.Errorf("%w: sort order must be nonnegative", ErrInvalid)
	}
	return in, nil
}

type Store struct {
	db  DB
	q   *dbsql.Queries
	now func() time.Time
}

// DB is what the store needs from a pool: queries and transactions.
type DB interface {
	dbsql.DBTX
	Begin(context.Context) (pgx.Tx, error)
}

func New(db DB, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, q: dbsql.New(db), now: now}
}

func stamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t.UTC(), Valid: true} }

func email(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// writeError maps database errors from inserts and updates.
func writeError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrDuplicate
		case "23503":
			return ErrInvalidReference
		case "23514":
			return fmt.Errorf("%w: rejected by database rule", ErrInvalid)
		}
	}
	return err
}

func deleteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23503" {
		return ErrInUse
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func yearOut(v dbsql.AcademicYear) Year {
	return Year{ID: v.ID, Label: v.Label, Active: v.IsActive, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
}

func classOut(v dbsql.Class, groups int64) Class {
	return Class{ID: v.ID, YearID: v.AcademicYearID, Code: v.Code, Name: v.Name, Active: v.IsActive, GroupCount: groups, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
}

func groupOut(v dbsql.StudyGroup) Group {
	g := Group{ID: v.ID, ClassID: v.ClassID, TopicID: v.TopicID, Name: v.Name, SortOrder: v.SortOrder, Active: v.IsActive, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
	if v.ExpectedEmail.Valid {
		e := v.ExpectedEmail.String
		g.Email = &e
	}
	return g
}

// ---- academic years ----------------------------------------------------------

// Years lists all academic years, newest first.
func (s *Store) Years(ctx context.Context) ([]Year, error) {
	rows, err := s.q.ListAcademicYears(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Year, 0, len(rows))
	for _, v := range rows {
		out = append(out, yearOut(v))
	}
	return out, nil
}

// CreateYear adds an inactive academic year.
func (s *Store) CreateYear(ctx context.Context, label string) (Year, error) {
	label, err := ValidateYearLabel(label)
	if err != nil {
		return Year{}, err
	}
	v, err := s.q.CreateAcademicYear(ctx, dbsql.CreateAcademicYearParams{Label: label, CreatedAt: stamp(s.now())})
	if err != nil {
		return Year{}, writeError(err)
	}
	return yearOut(v), nil
}

// ActivateYear makes id the only active academic year. The previous one is
// switched off first in the same transaction, so there is never a moment with
// two active years, and a missing id leaves the current one active.
func (s *Store) ActivateYear(ctx context.Context, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	q := s.q.WithTx(tx)
	now := stamp(s.now())
	ok, err := q.AcademicYearExists(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	if err := q.DeactivateAcademicYears(ctx, now); err != nil {
		return err
	}
	if _, err := q.ActivateAcademicYear(ctx, dbsql.ActivateAcademicYearParams{ID: id, UpdatedAt: now}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---- classes -----------------------------------------------------------------

func (s *Store) Classes(ctx context.Context, yearID int64) ([]Class, error) {
	rows, err := s.q.ListClasses(ctx, yearID)
	if err != nil {
		return nil, err
	}
	out := make([]Class, 0, len(rows))
	for _, v := range rows {
		out = append(out, classOut(dbsql.Class{ID: v.ID, AcademicYearID: v.AcademicYearID, Code: v.Code, Name: v.Name, IsActive: v.IsActive, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, v.GroupCount))
	}
	return out, nil
}

func (s *Store) Class(ctx context.Context, id int64) (Class, error) {
	v, err := s.q.GetClass(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Class{}, ErrNotFound
	}
	if err != nil {
		return Class{}, err
	}
	return classOut(dbsql.Class{ID: v.ID, AcademicYearID: v.AcademicYearID, Code: v.Code, Name: v.Name, IsActive: v.IsActive, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, v.GroupCount), nil
}

func (s *Store) checkYear(ctx context.Context, id int64) error {
	ok, err := s.q.AcademicYearExists(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidReference
	}
	return nil
}

func (s *Store) CreateClass(ctx context.Context, in ClassInput) (Class, error) {
	in, err := ValidateClass(in)
	if err != nil {
		return Class{}, err
	}
	if err := s.checkYear(ctx, in.YearID); err != nil {
		return Class{}, err
	}
	v, err := s.q.CreateClass(ctx, dbsql.CreateClassParams{AcademicYearID: in.YearID, Code: in.Code, Name: in.Name, CreatedAt: stamp(s.now())})
	if err != nil {
		return Class{}, writeError(err)
	}
	return classOut(v, 0), nil
}

// UpdateClass changes a class only if it still has updatedAt == expected.
func (s *Store) UpdateClass(ctx context.Context, id int64, expected time.Time, in ClassInput) (Class, error) {
	in, err := ValidateClass(in)
	if err != nil {
		return Class{}, err
	}
	if err := s.checkYear(ctx, in.YearID); err != nil {
		return Class{}, err
	}
	v, err := s.q.UpdateClass(ctx, dbsql.UpdateClassParams{ID: id, UpdatedAt: stamp(expected), AcademicYearID: in.YearID, Code: in.Code, Name: in.Name, IsActive: in.Active, UpdatedAt_2: stamp(s.now())})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, gerr := s.q.GetClass(ctx, id); errors.Is(gerr, pgx.ErrNoRows) {
			return Class{}, ErrNotFound
		} else if gerr != nil {
			return Class{}, gerr
		}
		return Class{}, ErrConflict
	}
	if err != nil {
		return Class{}, writeError(err)
	}
	return s.Class(ctx, v.ID)
}

// DeleteClass removes a class with no groups and returns its code.
func (s *Store) DeleteClass(ctx context.Context, id int64) (string, error) {
	code, err := s.q.DeleteClass(ctx, id)
	if err != nil {
		return "", deleteError(err)
	}
	return code, nil
}

// ---- groups ------------------------------------------------------------------

// Groups lists the groups of a class in display order.
func (s *Store) Groups(ctx context.Context, classID int64) ([]Group, error) {
	rows, err := s.q.ListGroups(ctx, classID)
	if err != nil {
		return nil, err
	}
	out := make([]Group, 0, len(rows))
	for _, v := range rows {
		out = append(out, groupOut(v))
	}
	return out, nil
}

func (s *Store) Group(ctx context.Context, id int64) (Group, error) {
	v, err := s.q.GetGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	return groupOut(v), nil
}

func (s *Store) checkGroupRefs(ctx context.Context, in GroupInput) error {
	ok, err := s.q.ClassExists(ctx, in.ClassID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidReference
	}
	ok, err = s.q.TopicExists(ctx, in.TopicID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidReference
	}
	return nil
}

func (s *Store) CreateGroup(ctx context.Context, in GroupInput) (Group, error) {
	in, err := ValidateGroup(in)
	if err != nil {
		return Group{}, err
	}
	if err := s.checkGroupRefs(ctx, in); err != nil {
		return Group{}, err
	}
	v, err := s.q.CreateGroup(ctx, dbsql.CreateGroupParams{ClassID: in.ClassID, TopicID: in.TopicID, Name: in.Name, ExpectedEmail: email(in.Email), SortOrder: in.SortOrder, CreatedAt: stamp(s.now())})
	if err != nil {
		return Group{}, writeError(err)
	}
	return groupOut(v), nil
}

// UpdateGroup changes a group only if it still has updatedAt == expected.
func (s *Store) UpdateGroup(ctx context.Context, id int64, expected time.Time, in GroupInput) (Group, error) {
	in, err := ValidateGroup(in)
	if err != nil {
		return Group{}, err
	}
	if err := s.checkGroupRefs(ctx, in); err != nil {
		return Group{}, err
	}
	v, err := s.q.UpdateGroup(ctx, dbsql.UpdateGroupParams{ID: id, UpdatedAt: stamp(expected), ClassID: in.ClassID, TopicID: in.TopicID, Name: in.Name, ExpectedEmail: email(in.Email), SortOrder: in.SortOrder, IsActive: in.Active, UpdatedAt_2: stamp(s.now())})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, gerr := s.q.GetGroup(ctx, id); errors.Is(gerr, pgx.ErrNoRows) {
			return Group{}, ErrNotFound
		} else if gerr != nil {
			return Group{}, gerr
		}
		return Group{}, ErrConflict
	}
	if err != nil {
		return Group{}, writeError(err)
	}
	return groupOut(v), nil
}

// DeleteGroup removes a group and returns its name. Once later parts add
// records that reference the group, the database refuses and ErrInUse is
// returned; the teacher deactivates the group instead.
func (s *Store) DeleteGroup(ctx context.Context, id int64) (string, error) {
	name, err := s.q.DeleteGroup(ctx, id)
	if err != nil {
		return "", deleteError(err)
	}
	return name, nil
}
