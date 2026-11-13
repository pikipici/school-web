// Package content validates and persists announcements and learning materials.
package content

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

type Kind uint8

const (
	Announcement Kind = iota
	Material
)

type Status string

const (
	Draft     Status = "draft"
	Published Status = "published"
)

var (
	ErrInvalid      = errors.New("invalid content")
	ErrInvalidTopic = errors.New("topic does not exist")
	ErrConflict     = errors.New("content update conflict")
	ErrNotFound     = errors.New("content not found")
)

type Input struct {
	Title     string
	Body      string
	Status    Status
	Pinned    bool
	TopicID   *int64
	SortOrder int32
}

type Item struct {
	ID          int64
	TopicID     *int64
	Title       string
	Body        string
	Status      Status
	Pinned      bool
	SortOrder   int32
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func Validate(kind Kind, in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	if n := utf8.RuneCountInString(in.Title); n < 1 || n > 160 {
		return Input{}, fmt.Errorf("%w: title must contain 1-160 characters", ErrInvalid)
	}
	if in.Status != Draft && in.Status != Published {
		return Input{}, fmt.Errorf("%w: unknown status", ErrInvalid)
	}
	switch kind {
	case Announcement:
		if utf8.RuneCountInString(in.Body) > 10000 {
			return Input{}, fmt.Errorf("%w: body too long", ErrInvalid)
		}
	case Material:
		if utf8.RuneCountInString(in.Body) > 20000 {
			return Input{}, fmt.Errorf("%w: description too long", ErrInvalid)
		}
		if in.SortOrder < 0 {
			return Input{}, fmt.Errorf("%w: sort order must be nonnegative", ErrInvalid)
		}
	default:
		return Input{}, fmt.Errorf("%w: unknown content kind", ErrInvalid)
	}
	return in, nil
}

type Store struct {
	q   *dbsql.Queries
	now func() time.Time
}

func New(db dbsql.DBTX, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{q: dbsql.New(db), now: now}
}

func stamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t.UTC(), Valid: true} }
func nullableID(id *int64) pgtype.Int8 {
	if id == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *id, Valid: true}
}
func ptrTime(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
func ptrID(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func announcementItem(v dbsql.Announcement) Item {
	return Item{ID: v.ID, Title: v.Title, Body: v.Body, Status: Status(v.Status), Pinned: v.Pinned, PublishedAt: ptrTime(v.PublishedAt), CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
}
func materialItem(v dbsql.Material) Item {
	return Item{ID: v.ID, TopicID: ptrID(v.TopicID), Title: v.Title, Body: v.Description, Status: Status(v.Status), SortOrder: v.SortOrder, PublishedAt: ptrTime(v.PublishedAt), CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
}

func (s *Store) CreateAnnouncement(ctx context.Context, in Input) (Item, error) {
	in, err := Validate(Announcement, in)
	if err != nil {
		return Item{}, err
	}
	v, err := s.q.CreateAnnouncement(ctx, dbsql.CreateAnnouncementParams{Title: in.Title, Body: in.Body, Status: string(in.Status), Pinned: in.Pinned, CreatedAt: stamp(s.now())})
	if err != nil {
		return Item{}, err
	}
	return announcementItem(v), nil
}
func (s *Store) CreateMaterial(ctx context.Context, in Input) (Item, error) {
	in, err := Validate(Material, in)
	if err != nil {
		return Item{}, err
	}
	if err = s.checkTopic(ctx, in.TopicID); err != nil {
		return Item{}, err
	}
	v, err := s.q.CreateMaterial(ctx, dbsql.CreateMaterialParams{TopicID: nullableID(in.TopicID), Title: in.Title, Description: in.Body, Status: string(in.Status), SortOrder: in.SortOrder, CreatedAt: stamp(s.now())})
	if err != nil {
		return Item{}, err
	}
	return materialItem(v), nil
}
func (s *Store) UpdateAnnouncement(ctx context.Context, id int64, expected time.Time, in Input) (Item, error) {
	in, err := Validate(Announcement, in)
	if err != nil {
		return Item{}, err
	}
	now := stamp(s.now())
	v, err := s.q.UpdateAnnouncement(ctx, dbsql.UpdateAnnouncementParams{ID: id, UpdatedAt: stamp(expected), Title: in.Title, Body: in.Body, Status: string(in.Status), Pinned: in.Pinned, UpdatedAt_2: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, s.missAnnouncement(ctx, id)
	}
	if err != nil {
		return Item{}, err
	}
	return announcementItem(v), nil
}
func (s *Store) UpdateMaterial(ctx context.Context, id int64, expected time.Time, in Input) (Item, error) {
	in, err := Validate(Material, in)
	if err != nil {
		return Item{}, err
	}
	if err = s.checkTopic(ctx, in.TopicID); err != nil {
		return Item{}, err
	}
	now := stamp(s.now())
	v, err := s.q.UpdateMaterial(ctx, dbsql.UpdateMaterialParams{ID: id, UpdatedAt: stamp(expected), TopicID: nullableID(in.TopicID), Title: in.Title, Description: in.Body, Status: string(in.Status), SortOrder: in.SortOrder, UpdatedAt_2: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, s.missMaterial(ctx, id)
	}
	if err != nil {
		return Item{}, err
	}
	return materialItem(v), nil
}
func (s *Store) checkTopic(ctx context.Context, id *int64) error {
	if id == nil {
		return nil
	}
	ok, err := s.q.TopicExists(ctx, *id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTopic
	}
	return nil
}
func (s *Store) missAnnouncement(ctx context.Context, id int64) error {
	_, err := s.q.GetAnnouncement(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrConflict
}
func (s *Store) missMaterial(ctx context.Context, id int64) error {
	_, err := s.q.GetMaterial(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrConflict
}
