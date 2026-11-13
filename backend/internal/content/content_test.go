package content

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

func TestValidateAndNormalize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		in        Input
		kind      Kind
		wantTitle string
		wantErr   bool
	}{
		{"trim title", Input{Title: "  Judul  ", Status: Draft}, Announcement, "Judul", false},
		{"unicode 160 runes", Input{Title: strings.Repeat("界", 160), Status: Draft}, Announcement, strings.Repeat("界", 160), false},
		{"unicode 161 runes", Input{Title: strings.Repeat("界", 161), Status: Draft}, Announcement, "", true},
		{"blank", Input{Title: " \n ", Status: Draft}, Announcement, "", true},
		{"bad status", Input{Title: "x", Status: "archived"}, Announcement, "", true},
		{"announcement body limit", Input{Title: "x", Body: strings.Repeat("a", 10001), Status: Draft}, Announcement, "", true},
		{"material description limit", Input{Title: "x", Body: strings.Repeat("a", 20001), Status: Draft}, Material, "", true},
		{"negative order", Input{Title: "x", Status: Draft, SortOrder: -1}, Material, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Validate(tc.kind, tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
			if got.Title != tc.wantTitle {
				t.Fatalf("title = %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestStoreLifecycleAndConflicts(t *testing.T) {
	pool := testdb.OpenMigrated(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 1, 2, 3, 0, time.UTC)
	s := New(pool, func() time.Time { return now })

	a, err := s.CreateAnnouncement(ctx, Input{Title: "  Halo 世界  ", Body: "isi", Status: Draft})
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Halo 世界" || a.PublishedAt != nil {
		t.Fatalf("bad create: %+v", a)
	}

	published, err := s.UpdateAnnouncement(ctx, a.ID, a.UpdatedAt, Input{Title: a.Title, Body: a.Body, Status: Published})
	if err != nil {
		t.Fatal(err)
	}
	if published.PublishedAt == nil || !published.PublishedAt.Equal(now) {
		t.Fatalf("published_at = %v", published.PublishedAt)
	}
	first := *published.PublishedAt

	now = now.Add(time.Hour)
	unpublished, err := s.UpdateAnnouncement(ctx, a.ID, published.UpdatedAt, Input{Title: a.Title, Body: a.Body, Status: Draft})
	if err != nil {
		t.Fatal(err)
	}
	republished, err := s.UpdateAnnouncement(ctx, a.ID, unpublished.UpdatedAt, Input{Title: a.Title, Body: a.Body, Status: Published})
	if err != nil {
		t.Fatal(err)
	}
	if republished.PublishedAt == nil || !republished.PublishedAt.Equal(first) {
		t.Fatalf("first publication not preserved: %v", republished.PublishedAt)
	}

	_, err = s.UpdateAnnouncement(ctx, a.ID, a.UpdatedAt, Input{Title: "stale", Status: Draft})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale error = %v", err)
	}
	_, err = s.UpdateAnnouncement(ctx, 999999, a.UpdatedAt, Input{Title: "missing", Status: Draft})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func TestMaterialTopicAndLifecycle(t *testing.T) {
	pool := testdb.OpenMigrated(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 4, 0, 0, 0, time.UTC)
	s := New(pool, func() time.Time { return now })
	var topicID int64
	if err := pool.QueryRow(ctx, `select id from topics where code='K1'`).Scan(&topicID); err != nil {
		t.Fatal(err)
	}

	m, err := s.CreateMaterial(ctx, Input{Title: " Materi ", Body: "deskripsi", Status: Published, TopicID: &topicID, SortOrder: 2})
	if err != nil {
		t.Fatal(err)
	}
	if m.PublishedAt == nil || !m.PublishedAt.Equal(now) || m.TopicID == nil || *m.TopicID != topicID {
		t.Fatalf("bad material: %+v", m)
	}

	missing := int64(999999)
	_, err = s.CreateMaterial(ctx, Input{Title: "x", Status: Draft, TopicID: &missing})
	if !errors.Is(err, ErrInvalidTopic) {
		t.Fatalf("topic error = %v", err)
	}

	_, err = s.UpdateMaterial(ctx, m.ID, m.UpdatedAt.Add(-time.Second), Input{Title: "x", Status: Draft})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale error = %v", err)
	}
	_, err = s.UpdateMaterial(ctx, 999999, m.UpdatedAt, Input{Title: "x", Status: Draft})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func TestAdminListsDetailsAndPermanentDeletes(t *testing.T) {
	pool := testdb.OpenMigrated(t)
	ctx := context.Background()
	s := New(pool, func() time.Time { return time.Date(2026, 11, 13, 6, 0, 0, 0, time.UTC) })
	a, err := s.CreateAnnouncement(ctx, Input{Title: "Draft internal", Body: "editable", Status: Draft, Pinned: true})
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.CreateMaterial(ctx, Input{Title: "Material draft", Body: "description", Status: Draft, SortOrder: 4})
	if err != nil {
		t.Fatal(err)
	}
	items, total, err := s.AdminAnnouncements(ctx, Page{Limit: 10})
	if err != nil || total != 1 || len(items) != 1 || items[0].Body != "editable" || items[0].Status != Draft {
		t.Fatalf("admin announcements: %#v %d %v", items, total, err)
	}
	got, err := s.AdminMaterial(ctx, m.ID)
	if err != nil || got.Body != "description" || got.SortOrder != 4 {
		t.Fatalf("admin material: %#v %v", got, err)
	}
	if title, err := s.DeleteAnnouncement(ctx, a.ID); err != nil || title != "Draft internal" {
		t.Fatalf("delete announcement: %q %v", title, err)
	}
	if title, err := s.DeleteMaterial(ctx, m.ID); err != nil || title != "Material draft" {
		t.Fatalf("delete material: %q %v", title, err)
	}
	if _, err := s.AdminAnnouncement(ctx, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted announcement: %v", err)
	}
	if _, err := s.DeleteMaterial(ctx, m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}
