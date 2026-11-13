package schema_test

import (
	"strings"
	"testing"
)

var contentTables = []string{"announcements", "topics", "materials"}

func TestContentTablesExist(t *testing.T) {
	pool := migrated(t)
	for _, name := range contentTables {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing after goose up", name)
		}
	}
}

func TestTopicsAreSeededInOrder(t *testing.T) {
	pool := migrated(t)

	rows, err := pool.Query(ctx(t), "select code, title from topics order by sort_order")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code, title string
		if err := rows.Scan(&code, &title); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(title) == "" {
			t.Errorf("topic %s has an empty title", code)
		}
		codes = append(codes, code)
	}
	if got := strings.Join(codes, ","); got != "K1,K2,K3,K4,K5,K6,K7" {
		t.Fatalf("seeded topics = %s; want K1..K7 in order", got)
	}

	var title string
	if err := pool.QueryRow(ctx(t), "select title from topics where code = 'K1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Struktur Data (Graf dan Tree)" {
		t.Fatalf("K1 title = %q", title)
	}

	mustFail(t, pool, "duplicate topic code", "insert into topics (code, title, sort_order) values ('K1', 'x', 99)")
	mustFail(t, pool, "bad topic code", "insert into topics (code, title, sort_order) values ('X1', 'x', 98)")
}

func TestAnnouncementConstraints(t *testing.T) {
	pool := migrated(t)

	mustExec(t, pool, "insert into announcements (title, body) values ('Libur', 'Sekolah libur.')")
	mustExec(t, pool, "insert into announcements (title, body, status, published_at, pinned) values ('Ujian', '', 'published', now(), true)")
	mustExec(t, pool, "insert into announcements (title, body) values ($1, $2)", strings.Repeat("é", 160), strings.Repeat("ü", 10000))

	var status string
	var pinned bool
	if err := pool.QueryRow(ctx(t), "select status, pinned from announcements where title = 'Libur'").Scan(&status, &pinned); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || pinned {
		t.Fatalf("defaults = %s/%v; want draft/false", status, pinned)
	}

	mustFail(t, pool, "empty title", "insert into announcements (title, body) values ('', 'x')")
	mustFail(t, pool, "whitespace title", "insert into announcements (title, body) values ('   ', 'x')")
	mustFail(t, pool, "title over 160 characters", "insert into announcements (title, body) values ($1, 'x')", strings.Repeat("a", 161))
	mustFail(t, pool, "body over 10000 characters", "insert into announcements (title, body) values ('x', $1)", strings.Repeat("a", 10001))
	mustFail(t, pool, "unknown status", "insert into announcements (title, body, status) values ('x', 'x', 'archived')")
	mustFail(t, pool, "published without published_at", "insert into announcements (title, body, status) values ('x', 'x', 'published')")
	mustFail(t, pool, "null body", "insert into announcements (title, body) values ('x', null)")
}

func TestMaterialConstraints(t *testing.T) {
	pool := migrated(t)

	var k1 int64
	if err := pool.QueryRow(ctx(t), "select id from topics where code = 'K1'").Scan(&k1); err != nil {
		t.Fatal(err)
	}

	mustExec(t, pool, "insert into materials (title, description, topic_id) values ('Graf dasar', 'Simpul dan sisi.', $1)", k1)
	mustExec(t, pool, "insert into materials (title) values ('Panduan pengumpulan')") // no topic = Umum
	mustExec(t, pool, "insert into materials (title, description) values ('x', $1)", strings.Repeat("a", 20000))

	var desc string
	var sortOrder int
	if err := pool.QueryRow(ctx(t), "select description, sort_order from materials where title = 'Panduan pengumpulan'").Scan(&desc, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if desc != "" || sortOrder != 0 {
		t.Fatalf("defaults = %q/%d; want empty description and sort_order 0", desc, sortOrder)
	}

	mustFail(t, pool, "unknown topic", "insert into materials (title, topic_id) values ('x', 999999)")
	mustFail(t, pool, "empty title", "insert into materials (title) values ('')")
	mustFail(t, pool, "description over 20000 characters", "insert into materials (title, description) values ('x', $1)", strings.Repeat("a", 20001))
	mustFail(t, pool, "published without published_at", "insert into materials (title, status) values ('x', 'published')")
	mustFail(t, pool, "negative sort order", "insert into materials (title, sort_order) values ('x', -1)")
	mustFail(t, pool, "deleting a topic still in use", "delete from topics where id = $1", k1)
}

func TestContentMigrationsRollBackWithoutTouchingAuthTables(t *testing.T) {
	pool := migrated(t)
	insertUser(t, pool, "fikri")

	goose(t, "down-to", "3")
	for _, name := range contentTables {
		if tableExists(t, pool, name) {
			t.Errorf("table %s still exists after down-to 3", name)
		}
	}
	var users int
	if err := pool.QueryRow(ctx(t), "select count(*) from admin_users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("admin_users rows after rollback = %d; want 1 (Fase 1 data untouched)", users)
	}

	goose(t, "up")
	var topics int
	if err := pool.QueryRow(ctx(t), "select count(*) from topics").Scan(&topics); err != nil {
		t.Fatal(err)
	}
	if topics != 7 {
		t.Fatalf("topics after re-up = %d; want 7 (seed not duplicated or lost)", topics)
	}
}
