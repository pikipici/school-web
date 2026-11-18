package progress_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

func findingInput() progress.FindingInput {
	return progress.FindingInput{FileType: "PPT", Location: "Slide 5", Problem: "Diagram kurang jelas", Action: "Tambahkan label simpul", Priority: "WAJIB"}
}

func TestFindingPublicationAndReviewRules(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	f, err := w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor)
	if err != nil {
		t.Fatal(err)
	}
	d, err := w.s.PublicDetail(ctx, w.group.ID)
	if err != nil || len(d.Findings) != 0 {
		t.Fatalf("draft leaked: %+v %v", d, err)
	}
	review, err := w.s.GetReview(ctx, w.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewNeedsRevision, w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("revision without published finding: %v", err)
	}
	if _, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewApproved, w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("approval without files: %v", err)
	}
	w.clk.Advance(time.Second)
	if err = w.s.PublishReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewChecking, w.actor); err != nil {
		t.Fatal(err)
	}
	review, _ = w.s.GetReview(ctx, w.group.ID)
	if err = w.s.PublishReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewChecking, w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("duplicate publish: %v", err)
	}
	d, err = w.s.PublicDetail(ctx, w.group.ID)
	if err != nil || len(d.Findings) != 1 || d.Findings[0].Location != "Slide 5" {
		t.Fatalf("published: %+v %v", d, err)
	}
	b, _ := json.Marshal(d)
	for _, secret := range []string{"email", "recordedBy", "subject", "internal", "createdBy", "note"} {
		if strings.Contains(string(b), "\""+secret+"\"") {
			t.Fatalf("private key %s: %s", secret, b)
		}
	}
	fs, err := w.s.Findings(ctx, w.group.ID)
	if err != nil || len(fs) != 1 {
		t.Fatal(err)
	}
	f = fs[0]
	if err = w.s.DeleteFinding(ctx, f.ID, f.UpdatedAt, w.actor); !errors.Is(err, progress.ErrPublished) {
		t.Fatalf("delete published: %v", err)
	}
	review, _ = w.s.GetReview(ctx, w.group.ID)
	w.clk.Advance(time.Second)
	review, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewNeedsRevision, w.actor)
	if err != nil {
		t.Fatal(err)
	}
	round := w.record(t, 0, progress.RoundInput{ReceivedAt: w.clk.now, PPT: true, PDF: true})
	if _, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewApproved, w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("approval with open finding: %v", err)
	}
	// Leave the revision state before resolving its last open mandatory finding.
	review, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewChecking, w.actor)
	if err != nil {
		t.Fatal(err)
	}
	w.clk.Advance(time.Second)
	f, err = w.s.UpdateFinding(ctx, f.ID, f.UpdatedAt, findingInput(), true, w.actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.s.UpdateFinding(ctx, f.ID, f.CreatedAt, findingInput(), false, w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	review, err = w.s.SetReview(ctx, w.group.ID, review.UpdatedAt, progress.ReviewApproved, w.actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.s.DeleteRound(ctx, round.ID, w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("approved files can be deleted: %v", err)
	}
	if w.publicRow(t).Complete {
		t.Fatal("complete without notebook")
	}
	if _, err = w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true}, w.actor); err != nil {
		t.Fatal(err)
	}
	if !w.publicRow(t).Complete {
		t.Fatal("expected complete")
	}
	// A later mandatory draft must not silently invalidate approval.
	if _, err = w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor); !errors.Is(err, progress.ErrReviewRule) {
		t.Fatalf("mandatory draft under approval: %v", err)
	}
	if _, err = w.s.DeleteGroup(ctx, w.group.ID); !errors.Is(err, progress.ErrInUse) {
		t.Fatalf("delete group with findings: %v", err)
	}
	if _, err = w.pool.Exec(ctx, "update study_groups set is_active=false where id=$1", w.group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = w.s.PublicDetail(ctx, w.group.ID); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("inactive detail: %v", err)
	}
}

func TestFindingDraftLifecycleValidationAndAtomicAudit(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	for _, in := range []progress.FindingInput{{}, {FileType: "EXE", Location: "x", Problem: "x", Action: "x", Priority: "WAJIB"}, {FileType: "PPT", Location: " ", Problem: "x", Action: "x", Priority: "WAJIB"}} {
		if _, err := w.s.CreateFinding(ctx, w.group.ID, in, w.actor); !errors.Is(err, progress.ErrInvalid) {
			t.Fatalf("invalid: %v", err)
		}
	}
	f, err := w.s.CreateFinding(ctx, w.group.ID, findingInput(), w.actor)
	if err != nil {
		t.Fatal(err)
	}
	w.clk.Advance(time.Second)
	in := findingInput()
	in.Location = "Slide 6"
	f, err = w.s.UpdateFinding(ctx, f.ID, f.UpdatedAt, in, false, w.actor)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.s.DeleteFinding(ctx, f.ID, f.CreatedAt, w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("stale delete: %v", err)
	}
	if err = w.s.DeleteFinding(ctx, f.ID, f.UpdatedAt, w.actor); err != nil {
		t.Fatal(err)
	}
	// Force an audit failure: the finding and timestamp must roll back together.
	_, err = w.pool.Exec(ctx, `CREATE FUNCTION reject_finding_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$; CREATE TRIGGER reject_finding_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION reject_finding_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.s.CreateFinding(ctx, w.group.ID, in, w.actor); err == nil {
		t.Fatal("expected audit failure")
	}
	fs, err := w.s.Findings(ctx, w.group.ID)
	if err != nil || len(fs) != 0 {
		t.Fatalf("mutation survived failed audit: %+v %v", fs, err)
	}
}
