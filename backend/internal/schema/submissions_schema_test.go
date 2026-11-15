package schema_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var submissionTables = []string{"submission_rounds", "group_notebooks"}

const insertRound = `insert into submission_rounds
	(group_id, round_number, received_at, sender_email, email_subject, ppt_received, pdf_received, ppt_filename, pdf_filename, internal_note, recorded_by)
	values ($1, $2, '2026-11-15 01:00:00+00', $3, $4, $5, $6, $7, $8, $9, $10)`

// round inserts a minimal valid round: PPT only, no metadata.
func round(t *testing.T, pool *pgxpool.Pool, group int64, n int) {
	t.Helper()
	mustExec(t, pool, insertRound, group, n, nil, "", true, false, "", "", "", nil)
}

func newGroup(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	y := activeYear(t, pool)
	var class int64
	if err := pool.QueryRow(ctx(t), "select id from classes where academic_year_id = $1 and code = '9A'", y).Scan(&class); err != nil {
		class = newClass(t, pool, y, "9A")
	}
	var id int64
	if err := pool.QueryRow(ctx(t), "insert into study_groups (class_id, topic_id, name) values ($1, $2, $3) returning id", class, topicID(t, pool, "K1"), name).Scan(&id); err != nil {
		t.Fatalf("insert group %q: %v", name, err)
	}
	return id
}

func TestSubmissionTablesExistAndAcceptValidRows(t *testing.T) {
	pool := migrated(t)
	for _, name := range submissionTables {
		if !tableExists(t, pool, name) {
			t.Fatalf("table %s missing after goose up", name)
		}
	}
	g := newGroup(t, pool, "Kelompok 1")
	admin := insertUser(t, pool, "fikri")

	mustExec(t, pool, insertRound, g, 0, "kelompok1.9a.2627.informatika@gmail.com", "Proyek Informatika IX - 9A - Kelompok 1", true, false, "Graf.pptx", "", "Hanya PPT", admin)
	mustExec(t, pool, insertRound, g, 1, nil, "REVISI 1 - Proyek", false, true, "", "Modul.pdf", "", admin)
	mustExec(t, pool, insertRound, g, 2, nil, "", true, true, "", "", "", nil)

	var n int
	var created bool
	if err := pool.QueryRow(ctx(t), "select count(*), bool_and(created_at is not null and updated_at is not null) from submission_rounds where group_id = $1", g).Scan(&n, &created); err != nil || n != 3 || !created {
		t.Fatalf("rounds = %d, timestamps = %v, err = %v", n, created, err)
	}

	mustExec(t, pool, "insert into group_notebooks (group_id, is_received, received_at, note, updated_by) values ($1, true, now(), 'Buku bersampul biru', $2)", g, admin)
	other := newGroup(t, pool, "Kelompok 2")
	mustExec(t, pool, "insert into group_notebooks (group_id) values ($1)", other)
	var received bool
	if err := pool.QueryRow(ctx(t), "select is_received from group_notebooks where group_id = $1", other).Scan(&received); err != nil || received {
		t.Fatalf("notebook default received = %v (%v), want false", received, err)
	}
}

func TestSubmissionRoundConstraints(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Kelompok 1")
	round(t, pool, g, 0)

	long := func(n int) string { return strings.Repeat("a", n) }
	for _, tc := range []struct {
		why  string
		args []any
	}{
		{"unknown group", []any{int64(999999), 1, nil, "", true, false, "", "", "", nil}},
		{"no file in round", []any{g, 1, nil, "", false, false, "", "", "", nil}},
		{"ppt name without ppt", []any{g, 1, nil, "", false, true, "a.pptx", "", "", nil}},
		{"pdf name without pdf", []any{g, 1, nil, "", true, false, "", "a.pdf", "", nil}},
		{"negative round", []any{g, -1, nil, "", true, false, "", "", "", nil}},
		{"duplicate round", []any{g, 0, nil, "", true, false, "", "", "", nil}},
		{"gap in rounds", []any{g, 2, nil, "", true, false, "", "", "", nil}},
		{"sender upper case", []any{g, 1, "Kelompok@Gmail.com", "", true, false, "", "", "", nil}},
		{"sender not an email", []any{g, 1, "kelompok", "", true, false, "", "", "", nil}},
		{"sender empty string", []any{g, 1, "", "", true, false, "", "", "", nil}},
		{"subject over 300", []any{g, 1, nil, long(301), true, false, "", "", "", nil}},
		{"subject control char", []any{g, 1, nil, "a\nb", true, false, "", "", "", nil}},
		{"file name with slash", []any{g, 1, nil, "", true, false, "dir/a.pptx", "", "", nil}},
		{"file name over 200", []any{g, 1, nil, "", true, false, long(201), "", "", nil}},
		{"note over 2000", []any{g, 1, nil, "", true, false, "", "", long(2001), nil}},
		{"unknown recorder", []any{g, 1, nil, "", true, false, "", "", "", int64(999999)}},
	} {
		mustFail(t, pool, tc.why, insertRound, tc.args...)
	}
	mustExec(t, pool, insertRound, g, 1, nil, long(300), true, true, long(200), long(200), long(2000), nil)

	// The first round of another group must be 0.
	g2 := newGroup(t, pool, "Kelompok 2")
	mustFail(t, pool, "first round not zero", insertRound, g2, 1, nil, "", true, false, "", "", "", nil)
	round(t, pool, g2, 0)
}

func TestOnlyOneRoundWinsUnderConcurrentInserts(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Kelompok 1")
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := pool.Exec(ctx(t), insertRound, g, 0, nil, "", true, false, "", "", "", nil); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("%d concurrent inserts of round 0 succeeded, want exactly 1", ok)
	}
}

func TestGroupNotebookConstraints(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Kelompok 1")
	const ins = "insert into group_notebooks (group_id, is_received, received_at, note) values ($1, $2, $3, $4)"
	for _, tc := range []struct {
		why  string
		args []any
	}{
		{"unknown group", []any{int64(999999), false, nil, ""}},
		{"received without time", []any{g, true, nil, ""}},
		{"time without received", []any{g, false, "2026-11-15 01:00:00+00", ""}},
		{"note over 1000", []any{g, false, nil, strings.Repeat("é", 1001)}},
	} {
		mustFail(t, pool, tc.why, ins, tc.args...)
	}
	mustExec(t, pool, ins, g, true, "2026-11-15 01:00:00+00", strings.Repeat("é", 1000))
	mustFail(t, pool, "second notebook row", ins, g, false, nil, "")
}

func TestGroupsWithRecordsCannotBeDeleted(t *testing.T) {
	pool := migrated(t)
	withRound := newGroup(t, pool, "Kelompok 1")
	withBook := newGroup(t, pool, "Kelompok 2")
	empty := newGroup(t, pool, "Kelompok 3")
	round(t, pool, withRound, 0)
	mustExec(t, pool, "insert into group_notebooks (group_id) values ($1)", withBook)

	mustFail(t, pool, "group with a round deleted", "delete from study_groups where id = $1", withRound)
	mustFail(t, pool, "group with a notebook row deleted", "delete from study_groups where id = $1", withBook)
	mustExec(t, pool, "delete from study_groups where id = $1", empty)
}

func TestDeletingAdminKeepsRecords(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Kelompok 1")
	admin := insertUser(t, pool, "fikri")
	mustExec(t, pool, insertRound, g, 0, nil, "", true, false, "", "", "", admin)
	mustExec(t, pool, "insert into group_notebooks (group_id, updated_by) values ($1, $2)", g, admin)
	mustExec(t, pool, "delete from admin_users where id = $1", admin)
	var rounds, books int
	if err := pool.QueryRow(ctx(t), "select (select count(*) from submission_rounds where recorded_by is null), (select count(*) from group_notebooks where updated_by is null)").Scan(&rounds, &books); err != nil || rounds != 1 || books != 1 {
		t.Fatalf("after admin delete: rounds=%d books=%d err=%v, want records kept with null actor", rounds, books, err)
	}
}

func TestSubmissionsDownAndUp(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Kelompok 1")
	round(t, pool, g, 0)

	goose(t, "down-to", "8")
	for _, name := range submissionTables {
		if tableExists(t, pool, name) {
			t.Errorf("table %s still exists after down-to 8", name)
		}
	}
	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from study_groups").Scan(&n); err != nil || n != 1 {
		t.Fatalf("groups after rollback = %d (%v); 3A data must be untouched", n, err)
	}
	goose(t, "up")
	for _, name := range submissionTables {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing after re-up", name)
		}
	}
}
