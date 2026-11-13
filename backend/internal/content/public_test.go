package content_test

import (
	"context"
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/content"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

func TestPublicReadsFilterOrderAndPaginate(t *testing.T) {
	pool := testdb.OpenMigrated(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
INSERT INTO announcements(title,body,status,pinned,published_at) VALUES
 ('draft secret','secret','draft',true,NULL),
 ('old pinned','a','published',true,'2025-01-01'),
 ('new normal','b','published',false,'2025-03-01'),
 ('new pinned','c','published',true,'2025-02-01');
INSERT INTO materials(topic_id,title,description,status,sort_order,published_at) VALUES
 ((SELECT id FROM topics WHERE code='K2'),'K2 first','x','published',0,'2025-01-01'),
 ((SELECT id FROM topics WHERE code='K1'),'draft material','secret','draft',0,NULL),
 ((SELECT id FROM topics WHERE code='K1'),'K1 second','y','published',2,'2025-01-01'),
 ((SELECT id FROM topics WHERE code='K1'),'K1 first','z','published',1,'2025-01-01');`)
	if err != nil {
		t.Fatal(err)
	}
	s := content.New(pool, nil)
	a, total, err := s.PublicAnnouncements(ctx, content.Page{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(a) != 2 || a[0].Title != "new pinned" || a[1].Title != "old pinned" {
		t.Fatalf("announcements = %#v total %d", a, total)
	}
	a2, _, err := s.PublicAnnouncements(ctx, content.Page{Limit: 2, Offset: 2})
	if err != nil || len(a2) != 1 || a2[0].Title != "new normal" {
		t.Fatalf("page 2 = %#v err %v", a2, err)
	}
	m, total, err := s.PublicMaterials(ctx, content.Page{Limit: 10}, "K1")
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(m) != 2 || m[0].Title != "K1 first" || m[1].Title != "K1 second" {
		t.Fatalf("materials = %#v total %d", m, total)
	}
}

func TestPublicDraftAndMissingDetailsAreBothNotFound(t *testing.T) {
	pool := testdb.OpenMigrated(t)
	ctx := context.Background()
	var aid, mid int64
	if err := pool.QueryRow(ctx, "INSERT INTO announcements(title,status) VALUES ('secret','draft') RETURNING id").Scan(&aid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO materials(title,status) VALUES ('secret','draft') RETURNING id").Scan(&mid); err != nil {
		t.Fatal(err)
	}
	s := content.New(pool, nil)
	for _, id := range []int64{aid, aid + 999} {
		if _, err := s.PublicAnnouncement(ctx, id); err != content.ErrNotFound {
			t.Fatalf("announcement %d: %v", id, err)
		}
	}
	for _, id := range []int64{mid, mid + 999} {
		if _, err := s.PublicMaterial(ctx, id); err != content.ErrNotFound {
			t.Fatalf("material %d: %v", id, err)
		}
	}
}
