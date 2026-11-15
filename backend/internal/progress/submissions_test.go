package progress_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
)

func adminID(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), "insert into admin_users (username, password_hash) values ('fikri', 'argon2id$x') returning id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

type world struct {
	s     *progress.Store
	pool  *pgxpool.Pool
	clk   *clock
	group progress.Group
	actor progress.Actor
}

func newWorld(t *testing.T) *world {
	t.Helper()
	s, pool, clk := setup(t)
	y := activeYear(t, s)
	c := mustClass(t, s, y.ID, "9A")
	g := mustGroup(t, s, c.ID, topic(t, pool, "K1"), "Kelompok 1", "", 0)
	return &world{s: s, pool: pool, clk: clk, group: g, actor: progress.Actor{UserID: adminID(t, pool), IP: "203.0.113.7"}}
}

func (w *world) record(t *testing.T, n int32, in progress.RoundInput) progress.Round {
	t.Helper()
	r, err := w.s.RecordRound(context.Background(), w.group.ID, n, in, w.actor)
	if err != nil {
		t.Fatalf("RecordRound %d: %v", n, err)
	}
	return r
}

func (w *world) publicRow(t *testing.T) progress.PublicGroup {
	t.Helper()
	p, err := w.s.PublicProgress(context.Background(), progress.PublicFilter{})
	if err != nil || len(p.Groups) != 1 {
		t.Fatalf("public = %+v, %v", p, err)
	}
	return p.Groups[0]
}

func ppt(at time.Time) progress.RoundInput { return progress.RoundInput{ReceivedAt: at, PPT: true} }
func pdf(at time.Time) progress.RoundInput { return progress.RoundInput{ReceivedAt: at, PDF: true} }

func TestSubmissionLaneFollowsRounds(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	at := w.clk.now.Add(-time.Hour)

	if got := w.publicRow(t).Submission; got != progress.SubmissionNone {
		t.Fatalf("no rounds = %s, want BELUM", got)
	}
	// AC02: PPT only -> SEBAGIAN.
	w.record(t, 0, progress.RoundInput{ReceivedAt: at, SenderEmail: " Kelompok1.9A@Gmail.com ", Subject: " Proyek Informatika IX ", PPT: true, PPTFile: " Graf.pptx "})
	if got := w.publicRow(t).Submission; got != progress.SubmissionPartial {
		t.Fatalf("after PPT = %s, want SEBAGIAN", got)
	}
	// AC03: PDF follows -> LENGKAP.
	w.record(t, 1, pdf(at.Add(time.Minute)))
	if got := w.publicRow(t).Submission; got != progress.SubmissionFull {
		t.Fatalf("after PDF = %s, want LENGKAP", got)
	}
	// AC08: a PPT-only revision keeps the earlier PDF.
	w.record(t, 2, ppt(at.Add(2*time.Minute)))
	row := w.publicRow(t)
	if row.Submission != progress.SubmissionFull || row.Complete {
		t.Fatalf("after PPT revision = %+v, want LENGKAP and not complete (review not approved)", row)
	}

	rounds, err := w.s.Rounds(ctx, w.group.ID)
	if err != nil || len(rounds) != 3 {
		t.Fatalf("Rounds = %d, %v", len(rounds), err)
	}
	r0 := rounds[0]
	if r0.Number != 0 || r0.SenderEmail == nil || *r0.SenderEmail != "kelompok1.9a@gmail.com" || r0.Subject != "Proyek Informatika IX" || r0.PPTFile != "Graf.pptx" || r0.RecordedBy == nil || *r0.RecordedBy != w.actor.UserID {
		t.Errorf("round 0 = %+v", r0)
	}
	if progress.SubmissionOf(rounds) != progress.SubmissionFull || progress.SubmissionOf(rounds[:1]) != progress.SubmissionPartial || progress.SubmissionOf(nil) != progress.SubmissionNone {
		t.Error("SubmissionOf disagrees with the public lane")
	}
	if _, err := w.s.Rounds(ctx, 999999); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("Rounds(missing) = %v", err)
	}
}

func TestRecordRoundRejectsDuplicatesGapsAndBadInput(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	at := w.clk.now.Add(-time.Hour)
	w.record(t, 0, ppt(at))

	if _, err := w.s.RecordRound(ctx, w.group.ID, 0, ppt(at), w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Errorf("double submit = %v, want ErrConflict", err)
	}
	if _, err := w.s.RecordRound(ctx, w.group.ID, 5, ppt(at), w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Errorf("gap = %v, want ErrConflict", err)
	}
	if _, err := w.s.RecordRound(ctx, 999999, 0, ppt(at), w.actor); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("missing group = %v, want ErrNotFound", err)
	}
	for name, in := range map[string]progress.RoundInput{
		"no file":            {ReceivedAt: at},
		"no time":            {PPT: true},
		"future":             {ReceivedAt: w.clk.now.Add(time.Hour), PPT: true},
		"ppt name, no ppt":   {ReceivedAt: at, PDF: true, PPTFile: "a.pptx"},
		"file name slash":    {ReceivedAt: at, PPT: true, PPTFile: "dir/a.pptx"},
		"bad sender":         {ReceivedAt: at, PPT: true, SenderEmail: "bukan email"},
		"subject control":    {ReceivedAt: at, PPT: true, Subject: "a\nb"},
		"subject over 300":   {ReceivedAt: at, PPT: true, Subject: strings.Repeat("a", 301)},
		"note over 2000":     {ReceivedAt: at, PPT: true, Note: strings.Repeat("é", 2001)},
		"file name over 200": {ReceivedAt: at, PDF: true, PDFFile: strings.Repeat("a", 201)},
	} {
		if _, err := w.s.RecordRound(ctx, w.group.ID, 1, in, w.actor); !errors.Is(err, progress.ErrInvalid) {
			t.Errorf("%s = %v, want ErrInvalid", name, err)
		}
	}
	// A few minutes of clock skew is allowed.
	w.record(t, 1, ppt(w.clk.now.Add(2*time.Minute)))
}

func TestUpdateAndDeleteRounds(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	at := w.clk.now.Add(-time.Hour)
	r0 := w.record(t, 0, ppt(at))
	r1 := w.record(t, 1, pdf(at))

	w.clk.Advance(time.Second)
	fixed, err := w.s.UpdateRound(ctx, r0.ID, r0.UpdatedAt, progress.RoundInput{ReceivedAt: at, PPT: true, PDF: true, Note: "salah centang"}, w.actor)
	if err != nil || !fixed.PDF || fixed.Note != "salah centang" || fixed.Number != 0 {
		t.Fatalf("UpdateRound = %+v, %v", fixed, err)
	}
	if _, err := w.s.UpdateRound(ctx, r0.ID, r0.UpdatedAt, ppt(at), w.actor); !errors.Is(err, progress.ErrConflict) {
		t.Errorf("stale update = %v", err)
	}
	if _, err := w.s.UpdateRound(ctx, 999999, r0.UpdatedAt, ppt(at), w.actor); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("missing update = %v", err)
	}
	if _, err := w.s.UpdateRound(ctx, r0.ID, fixed.UpdatedAt, progress.RoundInput{ReceivedAt: at}, w.actor); !errors.Is(err, progress.ErrInvalid) {
		t.Errorf("uncheck both = %v", err)
	}

	if _, err := w.s.DeleteRound(ctx, r0.ID, w.actor); !errors.Is(err, progress.ErrNotLast) {
		t.Errorf("delete first of two = %v, want ErrNotLast", err)
	}
	if _, err := w.s.DeleteGroup(ctx, w.group.ID); !errors.Is(err, progress.ErrInUse) {
		t.Errorf("delete group with rounds = %v, want ErrInUse", err)
	}
	gone, err := w.s.DeleteRound(ctx, r1.ID, w.actor)
	if err != nil || gone.Number != 1 {
		t.Fatalf("DeleteRound last = %+v, %v", gone, err)
	}
	if _, err := w.s.DeleteRound(ctx, r1.ID, w.actor); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("delete again = %v", err)
	}
	// After deleting round 1 the next number is 1 again.
	w.record(t, 1, pdf(at))
}

func TestNotebook(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()

	nb, err := w.s.Notebook(ctx, w.group.ID)
	if err != nil || nb.Received || nb.ReceivedAt != nil {
		t.Fatalf("default notebook = %+v, %v", nb, err)
	}
	nb, err = w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true, Note: " sampul biru "}, w.actor)
	if err != nil || !nb.Received || nb.ReceivedAt == nil || !nb.ReceivedAt.Equal(w.clk.now) || nb.Note != "sampul biru" {
		t.Fatalf("receive = %+v, %v", nb, err)
	}
	if got := w.publicRow(t).Notebook; got != progress.NotebookReceived {
		t.Errorf("public notebook = %s", got)
	}
	when := w.clk.now.Add(-24 * time.Hour)
	nb, err = w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true, ReceivedAt: &when}, w.actor)
	if err != nil || !nb.ReceivedAt.Equal(when) {
		t.Fatalf("receive with date = %+v, %v", nb, err)
	}
	nb, err = w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: false, ReceivedAt: &when}, w.actor)
	if err != nil || nb.Received || nb.ReceivedAt != nil {
		t.Fatalf("un-receive = %+v, %v", nb, err)
	}
	future := w.clk.now.Add(time.Hour)
	for name, in := range map[string]progress.NotebookInput{
		"future":    {Received: true, ReceivedAt: &future},
		"long note": {Note: strings.Repeat("é", 1001)},
	} {
		if _, err := w.s.SetNotebook(ctx, w.group.ID, in, w.actor); !errors.Is(err, progress.ErrInvalid) {
			t.Errorf("%s = %v, want ErrInvalid", name, err)
		}
	}
	if _, err := w.s.SetNotebook(ctx, 999999, progress.NotebookInput{}, w.actor); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("missing group = %v", err)
	}
	if _, err := w.s.Notebook(ctx, 999999); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("missing group notebook = %v", err)
	}
}

func TestSubmissionAuditIsWrittenInTheSameTransaction(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	at := w.clk.now.Add(-time.Hour)
	r0 := w.record(t, 0, progress.RoundInput{ReceivedAt: at, PPT: true, SenderEmail: "rahasia@gmail.com", Subject: "Subjek rahasia", Note: "catatan rahasia"})
	w.clk.Advance(time.Second)
	if _, err := w.s.UpdateRound(ctx, r0.ID, r0.UpdatedAt, progress.RoundInput{ReceivedAt: at, PPT: true, PDF: true}, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true}, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.DeleteRound(ctx, r0.ID, w.actor); err != nil {
		t.Fatal(err)
	}

	rows, err := w.pool.Query(ctx, "select action, actor_user_id, host(ip), detail from audit_log order by id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action, ip string
		var actor int64
		var detail map[string]any
		if err := rows.Scan(&action, &actor, &ip, &detail); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, action)
		raw, _ := json.Marshal(detail)
		if actor != w.actor.UserID || ip != "203.0.113.7" || detail["entityId"] == nil || detail["title"] != "Kelompok 1" || strings.Contains(string(raw), "rahasia") {
			t.Errorf("%s: actor=%d ip=%s detail=%s", action, actor, ip, raw)
		}
	}
	want := "[progress.round_recorded progress.round_updated progress.notebook_updated progress.round_deleted]"
	if fmtList(actions) != want {
		t.Errorf("audit = %v, want %s", actions, want)
	}

	// If the audit row cannot be written, the change itself is rolled back.
	for _, stmt := range []string{
		`create function fail_audit() returns trigger language plpgsql as $$ begin raise exception 'audit down'; end; $$`,
		`create trigger fail_audit before insert on audit_log for each row execute function fail_audit()`,
	} {
		if _, err := w.pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.s.RecordRound(ctx, w.group.ID, 0, ppt(at), w.actor); err == nil {
		t.Fatal("RecordRound succeeded although the audit write failed")
	}
	if _, err := w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: false}, w.actor); err == nil {
		t.Fatal("SetNotebook succeeded although the audit write failed")
	}
	var rounds int
	var received bool
	if err := w.pool.QueryRow(ctx, "select (select count(*) from submission_rounds), (select is_received from group_notebooks where group_id = $1)", w.group.ID).Scan(&rounds, &received); err != nil {
		t.Fatal(err)
	}
	if rounds != 0 || !received {
		t.Fatalf("after failed audit: rounds=%d received=%v, want 0 and unchanged true", rounds, received)
	}
}

func fmtList(xs []string) string { return "[" + strings.Join(xs, " ") + "]" }

func TestPublicUpdatedAtFollowsSubmissionChanges(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	before := w.publicRow(t).UpdatedAt
	w.clk.Advance(time.Hour)
	r := w.record(t, 0, ppt(w.clk.now))
	afterRecord := w.publicRow(t).UpdatedAt
	if !afterRecord.After(before) || !afterRecord.Equal(w.clk.now) {
		t.Fatalf("updatedAt after record = %v (before %v, now %v)", afterRecord, before, w.clk.now)
	}
	w.clk.Advance(time.Hour)
	if _, err := w.s.DeleteRound(ctx, r.ID, w.actor); err != nil {
		t.Fatal(err)
	}
	if got := w.publicRow(t).UpdatedAt; !got.Equal(w.clk.now) {
		t.Fatalf("updatedAt after delete = %v, want %v", got, w.clk.now)
	}
}

func TestDeleteGroupIgnoresAnEmptyNotebookRecord(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()

	// Ticked then unticked, no note: nothing real is left, so delete works.
	if _, err := w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: true}, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.SetNotebook(ctx, w.group.ID, progress.NotebookInput{Received: false}, w.actor); err != nil {
		t.Fatal(err)
	}
	if name, err := w.s.DeleteGroup(ctx, w.group.ID); err != nil || name != "Kelompok 1" {
		t.Fatalf("delete group with an empty notebook row = %q, %v", name, err)
	}
	var n int
	if err := w.pool.QueryRow(ctx, "select count(*) from group_notebooks").Scan(&n); err != nil || n != 0 {
		t.Fatalf("notebook rows left = %d (%v)", n, err)
	}

	// A received notebook, or one with a note, is a real record: refuse.
	c, err := w.s.Class(ctx, w.group.ClassID)
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []progress.NotebookInput{{Received: true}, {Received: false, Note: "belum lengkap"}} {
		g := mustGroup(t, w.s, c.ID, topic(t, w.pool, "K2"), "Kelompok "+strings.Repeat("x", len(in.Note)+1), "", 0)
		if _, err := w.s.SetNotebook(ctx, g.ID, in, w.actor); err != nil {
			t.Fatal(err)
		}
		if _, err := w.s.DeleteGroup(ctx, g.ID); !errors.Is(err, progress.ErrInUse) {
			t.Errorf("delete group with notebook %+v = %v, want ErrInUse", in, err)
		}
	}

	// Rounds still block, and a refused delete keeps the empty notebook row.
	g := mustGroup(t, w.s, c.ID, topic(t, w.pool, "K3"), "Kelompok Putaran", "", 0)
	w.group = g
	w.record(t, 0, ppt(w.clk.now.Add(-time.Hour)))
	if _, err := w.s.SetNotebook(ctx, g.ID, progress.NotebookInput{Received: false}, w.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := w.s.DeleteGroup(ctx, g.ID); !errors.Is(err, progress.ErrInUse) {
		t.Errorf("delete group with a round = %v, want ErrInUse", err)
	}
	if err := w.pool.QueryRow(ctx, "select count(*) from group_notebooks where group_id = $1", g.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("refused delete removed the notebook row (n=%d, %v)", n, err)
	}
	if _, err := w.s.DeleteGroup(ctx, 999999); !errors.Is(err, progress.ErrNotFound) {
		t.Errorf("delete missing group = %v", err)
	}
}
