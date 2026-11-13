package schema_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const insertAttachment = `insert into material_attachments
	(material_id, storage_key, original_name, kind, content_type, size_bytes, sha256)
	values ($1, $2, $3, $4, $5, $6, $7)`

const (
	mimePDF  = "application/pdf"
	mimePPT  = "application/vnd.ms-powerpoint"
	mimePPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
)

func newMaterial(t *testing.T, pool *pgxpool.Pool, title string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx(t), "insert into materials (title) values ($1) returning id", title).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func digest(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

func key(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

func TestAttachmentTableExistsAndAcceptsValidRows(t *testing.T) {
	pool := migrated(t)
	if !tableExists(t, pool, "material_attachments") {
		t.Fatal("material_attachments missing after goose up")
	}
	m := newMaterial(t, pool, "Graf")

	mustExec(t, pool, insertAttachment, m, key(1), "Modul Graf.pdf", "pdf", mimePDF, int64(1), digest("a"))
	mustExec(t, pool, insertAttachment, m, key(2), "Slide lama.ppt", "ppt", mimePPT, int64(1024), digest("b"))
	mustExec(t, pool, insertAttachment, m, key(3), "Slide baru.pptx", "pptx", mimePPTX, int64(25*1024*1024), digest("c"))

	var created bool
	if err := pool.QueryRow(ctx(t), "select created_at is not null from material_attachments where storage_key = $1", key(1)).Scan(&created); err != nil || !created {
		t.Fatalf("created_at default missing: %v", err)
	}
}

func TestAttachmentConstraints(t *testing.T) {
	pool := migrated(t)
	m := newMaterial(t, pool, "Graf")
	mustExec(t, pool, insertAttachment, m, key(1), "a.pdf", "pdf", mimePDF, int64(10), digest("a"))

	cases := []struct {
		why  string
		args []any
	}{
		{"unknown material", []any{int64(999999), key(2), "a.pdf", "pdf", mimePDF, int64(10), digest("x")}},
		{"duplicate storage key", []any{m, key(1), "b.pdf", "pdf", mimePDF, int64(10), digest("y")}},
		{"storage key not a UUID", []any{m, "../../etc/passwd", "b.pdf", "pdf", mimePDF, int64(10), digest("y")}},
		{"unknown kind", []any{m, key(3), "a.docx", "docx", mimePDF, int64(10), digest("y")}},
		{"macro-enabled kind", []any{m, key(3), "a.pptm", "pptm", "application/vnd.ms-powerpoint.presentation.macroEnabled.12", int64(10), digest("y")}},
		{"kind and content type disagree", []any{m, key(3), "a.pdf", "pdf", mimePPTX, int64(10), digest("y")}},
		{"empty file", []any{m, key(3), "a.pdf", "pdf", mimePDF, int64(0), digest("y")}},
		{"over 25 MiB", []any{m, key(3), "a.pdf", "pdf", mimePDF, int64(25*1024*1024 + 1), digest("y")}},
		{"hash wrong length", []any{m, key(3), "a.pdf", "pdf", mimePDF, int64(10), []byte("short")}},
		{"empty name", []any{m, key(3), "", "pdf", mimePDF, int64(10), digest("y")}},
		{"name with slash", []any{m, key(3), "dir/a.pdf", "pdf", mimePDF, int64(10), digest("y")}},
		{"name with backslash", []any{m, key(3), `dir\a.pdf`, "pdf", mimePDF, int64(10), digest("y")}},
		{"name with control character", []any{m, key(3), "a\n.pdf", "pdf", mimePDF, int64(10), digest("y")}},
		{"name over 200 characters", []any{m, key(3), strings.Repeat("é", 197) + ".pdf", "pdf", mimePDF, int64(10), digest("y")}},
	}
	for _, c := range cases {
		mustFail(t, pool, c.why, insertAttachment, c.args...)
	}
	mustExec(t, pool, insertAttachment, m, key(4), strings.Repeat("é", 196)+".pdf", "pdf", mimePDF, int64(10), digest("z"))
}

func TestAtMostFiveAttachmentsPerMaterial(t *testing.T) {
	pool := migrated(t)
	m := newMaterial(t, pool, "Graf")
	other := newMaterial(t, pool, "Tree")

	for i := 1; i <= 5; i++ {
		mustExec(t, pool, insertAttachment, m, key(i), fmt.Sprintf("%d.pdf", i), "pdf", mimePDF, int64(10), digest(fmt.Sprint(i)))
	}
	mustFail(t, pool, "sixth attachment", insertAttachment, m, key(6), "6.pdf", "pdf", mimePDF, int64(10), digest("6"))
	mustExec(t, pool, insertAttachment, other, key(7), "1.pdf", "pdf", mimePDF, int64(10), digest("7"))

	mustExec(t, pool, "delete from material_attachments where storage_key = $1", key(1))
	mustExec(t, pool, insertAttachment, m, key(8), "6.pdf", "pdf", mimePDF, int64(10), digest("8"))
}

func TestFiveLimitHoldsUnderConcurrentInserts(t *testing.T) {
	pool := migrated(t)
	m := newMaterial(t, pool, "Graf")

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 1; i <= 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := pool.Exec(ctx(t), insertAttachment, m, key(100+i), fmt.Sprintf("%d.pdf", i), "pdf", mimePDF, int64(10), digest(fmt.Sprint(i)))
			if err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from material_attachments where material_id = $1", m).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 5 || ok != 5 {
		t.Fatalf("concurrent inserts stored %d rows (%d succeeded); want exactly 5", n, ok)
	}
}

func TestDeletingMaterialRemovesItsAttachmentRows(t *testing.T) {
	pool := migrated(t)
	m := newMaterial(t, pool, "Graf")
	mustExec(t, pool, insertAttachment, m, key(1), "a.pdf", "pdf", mimePDF, int64(10), digest("a"))

	var keys []string
	rows, err := pool.Query(ctx(t), "delete from materials where id = $1 returning (select array_agg(storage_key::text) from material_attachments where material_id = $1)", m)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		if err := rows.Scan(&keys); err != nil {
			t.Fatal(err)
		}
	}
	rows.Close()
	if len(keys) != 1 || keys[0] != key(1) {
		t.Fatalf("storage keys visible at delete time = %v; the app needs them to remove files", keys)
	}

	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from material_attachments").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("attachment rows after material delete = %d; want 0", n)
	}
}

func TestAttachmentMigrationRollsBackCleanly(t *testing.T) {
	pool := migrated(t)
	newMaterial(t, pool, "tetap ada")

	goose(t, "down-to", "6")
	if tableExists(t, pool, "material_attachments") {
		t.Fatal("material_attachments still exists after down-to 6")
	}
	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from materials").Scan(&n); err != nil || n != 1 {
		t.Fatalf("materials after rollback = %d (%v); Fase 2A data must be untouched", n, err)
	}
	goose(t, "up")
	if !tableExists(t, pool, "material_attachments") {
		t.Fatal("material_attachments missing after re-up")
	}
}
