package content

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// AdminStore is the editable content surface used by authenticated handlers.
type AdminStore interface {
	AdminAnnouncements(context.Context, Page) ([]Item, int64, error)
	AdminAnnouncement(context.Context, int64) (Item, error)
	CreateAnnouncement(context.Context, Input) (Item, error)
	UpdateAnnouncement(context.Context, int64, time.Time, Input) (Item, error)
	DeleteAnnouncement(context.Context, int64) (string, error)
	AdminMaterials(context.Context, Page) ([]Item, int64, error)
	AdminMaterial(context.Context, int64) (Item, error)
	CreateMaterial(context.Context, Input) (Item, error)
	UpdateMaterial(context.Context, int64, time.Time, Input) (Item, error)
	DeleteMaterial(context.Context, int64) (string, error)
}

func (s *Store) AdminAnnouncements(ctx context.Context, p Page) ([]Item, int64, error) {
	rows, err := s.q.ListAdminAnnouncements(ctx, dbsql.ListAdminAnnouncementsParams{Limit: p.Limit, Offset: p.Offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountAdminAnnouncements(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, announcementItem(row))
	}
	return out, total, nil
}
func (s *Store) AdminAnnouncement(ctx context.Context, id int64) (Item, error) {
	v, err := s.q.GetAnnouncement(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	return announcementItem(v), nil
}
func (s *Store) DeleteAnnouncement(ctx context.Context, id int64) (string, error) {
	title, err := s.q.DeleteAnnouncement(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return title, err
}
func (s *Store) AdminMaterials(ctx context.Context, p Page) ([]Item, int64, error) {
	rows, err := s.q.ListAdminMaterials(ctx, dbsql.ListAdminMaterialsParams{Limit: p.Limit, Offset: p.Offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountAdminMaterials(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, materialItem(row))
	}
	return out, total, nil
}
func (s *Store) AdminMaterial(ctx context.Context, id int64) (Item, error) {
	v, err := s.q.GetMaterial(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	return materialItem(v), nil
}
func (s *Store) DeleteMaterial(ctx context.Context, id int64) (string, error) {
	title, err := s.q.DeleteMaterial(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return title, err
}
