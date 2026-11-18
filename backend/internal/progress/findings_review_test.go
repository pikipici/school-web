package progress_test

import (
	"context"
	"errors"
	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
	"testing"
	"time"
)

func TestPublishedFindingContentCannotChange(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	f, err := w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := w.s.GetReview(ctx, w.group.ID)
	if err = w.s.PublishReview(ctx, w.group.ID, r.UpdatedAt, progress.ReviewChecking, w.actor); err != nil {
		t.Fatal(err)
	}
	fs, _ := w.s.Findings(ctx, w.group.ID)
	f = fs[0]
	for _, field := range []string{"file", "location", "problem", "action", "priority"} {
		in := findingInput()
		switch field {
		case "file":
			in.FileType = "PDF"
		case "location":
			in.Location = "Halaman 1"
		case "problem":
			in.Problem = "Lain"
		case "action":
			in.Action = "Ganti"
		case "priority":
			in.Priority = "SARAN"
		}
		if _, err = w.s.UpdateFinding(ctx, f.ID, f.UpdatedAt, in, false, w.actor); !errors.Is(err, progress.ErrPublished) {
			t.Fatalf("published %s changed: %v", field, err)
		}
	}
	w.clk.Advance(time.Second)
	if _, err = w.s.UpdateFinding(ctx, f.ID, f.UpdatedAt, findingInput(), true, w.actor); err != nil {
		t.Fatalf("resolve: %v", err)
	}
}

func TestPublishReviewIsAtomic(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	_, err := w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := w.s.GetReview(ctx, w.group.ID)
	if err = w.s.PublishReview(ctx, w.group.ID, r.UpdatedAt.Add(-time.Second), progress.ReviewNeedsRevision, w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("stale publish: %v", err)
	}
	d, _ := w.s.PublicDetail(ctx, w.group.ID)
	if len(d.Findings) != 0 || d.Review != progress.ReviewNotChecked {
		t.Fatal("partial publish on conflict")
	}
	if err = w.s.PublishReview(ctx, w.group.ID, r.UpdatedAt, progress.ReviewApproved, w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("invalid approval: %v", err)
	}
	d, _ = w.s.PublicDetail(ctx, w.group.ID)
	if len(d.Findings) != 0 {
		t.Fatal("publication survived failed validation")
	}
	_, err = w.pool.Exec(ctx, `CREATE FUNCTION fail_publish_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$; CREATE TRIGGER fail_publish_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_publish_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.s.PublishReview(ctx, w.group.ID, r.UpdatedAt, progress.ReviewNeedsRevision, w.actor); err == nil {
		t.Fatal("expected audit failure")
	}
	d, _ = w.s.PublicDetail(ctx, w.group.ID)
	if len(d.Findings) != 0 || d.Review != progress.ReviewNotChecked {
		t.Fatal("partial publish on audit failure")
	}
	_, err = w.pool.Exec(ctx, `DROP TRIGGER fail_publish_audit ON audit_log`)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.s.PublishReview(ctx, w.group.ID, r.UpdatedAt, progress.ReviewNeedsRevision, w.actor); err != nil {
		t.Fatal(err)
	}
	d, _ = w.s.PublicDetail(ctx, w.group.ID)
	if len(d.Findings) != 1 || d.Review != progress.ReviewNeedsRevision {
		t.Fatal("publication and status must commit together")
	}
}
