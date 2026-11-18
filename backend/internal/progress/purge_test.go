package progress_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

func count(t *testing.T, w *world, sql string, id int64) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(context.Background(), sql, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// fill gives the group a round, a received notebook, and a published finding:
// everything that blocks a normal delete.
func fill(t *testing.T, w *world) {
	t.Helper()
	ctx := context.Background()
	w.record(t, 0, ppt(w.clk.now.Add(-time.Hour)))
	if _, err := w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true, Note: "ada"}, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor); err != nil {
		t.Fatal(err)
	}
	review, err := w.s.GetReview(ctx, w.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	w.clk.Advance(time.Second)
	if err := w.s.PublishReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewNeedsRevision, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.DeleteGroup(ctx, w.group.ID); !errors.Is(err, progress.ErrInUse) {
		t.Fatalf("plain delete of filled group = %v, want ErrInUse", err)
	}
}

func remaining(t *testing.T, w *world) int {
	id := w.group.ID
	return count(t, w, "select count(*) from study_groups where id = $1", id) +
		count(t, w, "select count(*) from submission_rounds where group_id = $1", id) +
		count(t, w, "select count(*) from group_notebooks where group_id = $1", id) +
		count(t, w, "select count(*) from evaluation_findings where group_id = $1", id)
}

func TestPurgeGroupRemovesEverything(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	fill(t, w)

	// Wrong confirmation keeps every row.
	for _, bad := range []string{"", "kelompok 1", "Kelompok 2"} {
		if _, err := w.s.PurgeGroup(ctx, w.group.ID, bad, w.actor); !errors.Is(err, progress.ErrConfirm) {
			t.Fatalf("confirm %q = %v, want ErrConfirm", bad, err)
		}
	}
	if got := remaining(t, w); got != 4 {
		t.Fatalf("rows after refused purge = %d, want 4", got)
	}

	p, err := w.s.PurgeGroup(ctx, w.group.ID, "  Kelompok 1 ", w.actor)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Kelompok 1" || p.Rounds != 1 || p.Findings != 1 || !p.Notebook {
		t.Fatalf("purge result = %+v", p)
	}
	if got := remaining(t, w); got != 0 {
		t.Fatalf("rows after purge = %d, want 0", got)
	}
	var detail string
	if err := w.pool.QueryRow(ctx, "select detail::text from audit_log where action = 'progress.group_purged'").Scan(&detail); err != nil {
		t.Fatalf("purge audit: %v", err)
	}
	for _, want := range []string{`"title": "Kelompok 1"`, `"rounds": "1"`, `"findings": "1"`} {
		if !strings.Contains(detail, want) {
			t.Fatalf("audit detail %s lacks %s", detail, want)
		}
	}
	if _, err := w.s.PurgeGroup(ctx, w.group.ID, "Kelompok 1", w.actor); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("purge again = %v, want ErrNotFound", err)
	}
}

func TestPurgeGroupIsAtomicWithAudit(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	fill(t, w)
	if _, err := w.pool.Exec(ctx, `CREATE FUNCTION fail_purge_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit down'; END $$; CREATE TRIGGER fail_purge_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_purge_audit()`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.PurgeGroup(ctx, w.group.ID, "Kelompok 1", w.actor); err == nil {
		t.Fatal("purge succeeded without audit")
	}
	if got := remaining(t, w); got != 4 {
		t.Fatalf("rows after failed purge = %d, want 4", got)
	}
}
