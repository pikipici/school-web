package content

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

type Page struct{ Limit, Offset int32 }
type PublicAnnouncement struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"publishedAt"`
}
type PublicTopic struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Title     string `json:"title"`
	SortOrder int32  `json:"sortOrder"`
}
type PublicMaterial struct {
	ID          int64                `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	PublishedAt time.Time            `json:"publishedAt"`
	Topic       *PublicMaterialTopic `json:"topic"`
}

type PublicMaterialTopic struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type PublicReader interface {
	PublicAnnouncements(context.Context, Page) ([]PublicAnnouncement, int64, error)
	PublicAnnouncement(context.Context, int64) (PublicAnnouncement, error)
	PublicTopics(context.Context) ([]PublicTopic, error)
	PublicMaterials(context.Context, Page, string) ([]PublicMaterial, int64, error)
	PublicMaterial(context.Context, int64) (PublicMaterial, error)
}

func pubAnnouncement(id int64, title, body string, published pgtype.Timestamptz) PublicAnnouncement {
	return PublicAnnouncement{id, title, body, published.Time}
}
func pubMaterial(id int64, title, description string, published pgtype.Timestamptz, topicID pgtype.Int8, code, topicTitle pgtype.Text) PublicMaterial {
	v := PublicMaterial{ID: id, Title: title, Description: description, PublishedAt: published.Time}
	if topicID.Valid {
		v.Topic = &PublicMaterialTopic{ID: topicID.Int64, Code: code.String, Title: topicTitle.String}
	}
	return v
}
func topicArg(code string) pgtype.Text { return pgtype.Text{String: code, Valid: code != ""} }

func (s *Store) PublicAnnouncements(ctx context.Context, p Page) ([]PublicAnnouncement, int64, error) {
	rows, err := s.q.ListPublicAnnouncements(ctx, dbsql.ListPublicAnnouncementsParams{Limit: p.Limit, Offset: p.Offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountPublicAnnouncements(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PublicAnnouncement, 0, len(rows))
	for _, v := range rows {
		out = append(out, pubAnnouncement(v.ID, v.Title, v.Body, v.PublishedAt))
	}
	return out, total, nil
}
func (s *Store) PublicAnnouncement(ctx context.Context, id int64) (PublicAnnouncement, error) {
	v, err := s.q.GetPublicAnnouncement(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicAnnouncement{}, ErrNotFound
	}
	if err != nil {
		return PublicAnnouncement{}, err
	}
	return pubAnnouncement(v.ID, v.Title, v.Body, v.PublishedAt), nil
}
func (s *Store) PublicTopics(ctx context.Context) ([]PublicTopic, error) {
	rows, err := s.q.ListPublicTopics(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PublicTopic, 0, len(rows))
	for _, v := range rows {
		out = append(out, PublicTopic{v.ID, v.Code, v.Title, v.SortOrder})
	}
	return out, nil
}
func (s *Store) PublicMaterials(ctx context.Context, p Page, code string) ([]PublicMaterial, int64, error) {
	arg := topicArg(code)
	rows, err := s.q.ListPublicMaterials(ctx, dbsql.ListPublicMaterialsParams{Limit: p.Limit, Offset: p.Offset, TopicCode: arg})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountPublicMaterials(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PublicMaterial, 0, len(rows))
	for _, v := range rows {
		out = append(out, pubMaterial(v.ID, v.Title, v.Description, v.PublishedAt, v.TopicID, v.TopicCode, v.TopicTitle))
	}
	return out, total, nil
}
func (s *Store) PublicMaterial(ctx context.Context, id int64) (PublicMaterial, error) {
	v, err := s.q.GetPublicMaterial(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicMaterial{}, ErrNotFound
	}
	if err != nil {
		return PublicMaterial{}, err
	}
	return pubMaterial(v.ID, v.Title, v.Description, v.PublishedAt, v.TopicID, v.TopicCode, v.TopicTitle), nil
}
