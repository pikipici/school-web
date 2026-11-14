package schema_test

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var progressTables = []string{"academic_years", "classes", "study_groups"}

func activeYear(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx(t), "select id from academic_years where is_active").Scan(&id); err != nil {
		t.Fatalf("active academic year: %v", err)
	}
	return id
}

func newClass(t *testing.T, pool *pgxpool.Pool, year int64, code string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx(t), "insert into classes (academic_year_id, code) values ($1, $2) returning id", year, code).Scan(&id); err != nil {
		t.Fatalf("insert class %q: %v", code, err)
	}
	return id
}

func topicID(t *testing.T, pool *pgxpool.Pool, code string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx(t), "select id from topics where code = $1", code).Scan(&id); err != nil {
		t.Fatalf("topic %s: %v", code, err)
	}
	return id
}

const insertGroup = `insert into study_groups (class_id, topic_id, name, expected_email, sort_order) values ($1, $2, $3, $4, $5)`

func TestProgressTablesExistAndSeedActiveYear(t *testing.T) {
	pool := migrated(t)
	for _, name := range progressTables {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing after goose up", name)
		}
	}
	var label string
	var n int
	if err := pool.QueryRow(ctx(t), "select min(label), count(*) from academic_years where is_active").Scan(&label, &n); err != nil {
		t.Fatal(err)
	}
	if n != 1 || label != "2026/2027" {
		t.Fatalf("active years = %d (%q), want exactly 2026/2027", n, label)
	}
}

func TestAcademicYearConstraints(t *testing.T) {
	pool := migrated(t)
	mustExec(t, pool, "insert into academic_years (label) values ('2027/2028')")
	var active bool
	if err := pool.QueryRow(ctx(t), "select is_active from academic_years where label = '2027/2028'").Scan(&active); err != nil || active {
		t.Fatalf("new year must default to inactive (active=%v, err=%v)", active, err)
	}
	for why, label := range map[string]string{
		"duplicate":       "2026/2027",
		"not consecutive": "2026/2028",
		"reversed":        "2027/2026",
		"wrong format":    "2026-2027",
		"short years":     "26/27",
		"trailing text":   "2026/2027 genap",
		"empty":           "",
		"leading space":   " 2028/2029",
	} {
		mustFail(t, pool, why, "insert into academic_years (label) values ($1)", label)
	}
	mustFail(t, pool, "second active year", "update academic_years set is_active = true where label = '2027/2028'")

	// Switching the active year in one transaction is allowed.
	mustExec(t, pool, `update academic_years set is_active = (label = '2027/2028')`)
	if got := activeYear(t, pool); got == 0 {
		t.Fatal("no active year after switch")
	}
}

func TestClassConstraints(t *testing.T) {
	pool := migrated(t)
	y := activeYear(t, pool)
	mustExec(t, pool, "insert into academic_years (label) values ('2027/2028')")
	var y2 int64
	if err := pool.QueryRow(ctx(t), "select id from academic_years where label = '2027/2028'").Scan(&y2); err != nil {
		t.Fatal(err)
	}
	c := newClass(t, pool, y, "9A")
	newClass(t, pool, y, "IX B")
	newClass(t, pool, y2, "9A") // same code in another year is fine

	var isActive bool
	var name string
	if err := pool.QueryRow(ctx(t), "select is_active, name from classes where id = $1", c).Scan(&isActive, &name); err != nil || !isActive || name != "" {
		t.Fatalf("class defaults: active=%v name=%q err=%v", isActive, name, err)
	}
	const ins = "insert into classes (academic_year_id, code, name) values ($1, $2, $3)"
	for _, tc := range []struct {
		why  string
		year any
		code string
		name string
	}{
		{"unknown year", int64(999999), "9C", ""},
		{"duplicate code", y, "9A", ""},
		{"duplicate code other case", y, "9a", ""},
		{"empty code", y, "", ""},
		{"blank code", y, "   ", ""},
		{"code over 20 characters", y, strings.Repeat("A", 21), ""},
		{"code with slash", y, "9/A", ""},
		{"code with control character", y, "9\nA", ""},
		{"name over 80 characters", y, "9D", strings.Repeat("n", 81)},
	} {
		mustFail(t, pool, tc.why, ins, tc.year, tc.code, tc.name)
	}
	mustExec(t, pool, ins, y, strings.Repeat("A", 20), strings.Repeat("n", 80))
	mustFail(t, pool, "year with classes deleted", "delete from academic_years where id = $1", y)
}

func TestStudyGroupConstraints(t *testing.T) {
	pool := migrated(t)
	y := activeYear(t, pool)
	c := newClass(t, pool, y, "9A")
	c2 := newClass(t, pool, y, "9B")
	k1 := topicID(t, pool, "K1")

	mustExec(t, pool, insertGroup, c, k1, "Kelompok 1", "kelompok1.9a.2627.informatika@gmail.com", 0)
	mustExec(t, pool, insertGroup, c, k1, "Kelompok 2", nil, 1) // same topic twice and no email are fine
	mustExec(t, pool, insertGroup, c2, k1, "Kelompok 1", nil, 0)

	var active bool
	if err := pool.QueryRow(ctx(t), "select is_active from study_groups where class_id = $1 and name = 'Kelompok 2'", c).Scan(&active); err != nil || !active {
		t.Fatalf("group must default to active (active=%v, err=%v)", active, err)
	}
	for _, tc := range []struct {
		why   string
		class any
		topic any
		name  string
		email any
		sort  int
	}{
		{"unknown class", int64(999999), k1, "Kelompok 9", nil, 0},
		{"missing topic", c, nil, "Kelompok 9", nil, 0},
		{"unknown topic", c, int64(999999), "Kelompok 9", nil, 0},
		{"duplicate name", c, k1, "Kelompok 1", nil, 0},
		{"duplicate name other case", c, k1, "kelompok 1", nil, 0},
		{"empty name", c, k1, "", nil, 0},
		{"blank name", c, k1, "   ", nil, 0},
		{"name over 60 characters", c, k1, strings.Repeat("é", 61), nil, 0},
		{"name with control character", c, k1, "Kel\tompok", nil, 0},
		{"email upper case", c, k1, "Kelompok 9", "Kelompok9@gmail.com", 0},
		{"email without at", c, k1, "Kelompok 9", "kelompok9.gmail.com", 0},
		{"email with space", c, k1, "Kelompok 9", "kel ompok9@gmail.com", 0},
		{"email empty string", c, k1, "Kelompok 9", "", 0},
		{"email over 254 characters", c, k1, "Kelompok 9", strings.Repeat("a", 245) + "@gmail.com", 0},
		{"negative sort order", c, k1, "Kelompok 9", nil, -1},
	} {
		mustFail(t, pool, tc.why, insertGroup, tc.class, tc.topic, tc.name, tc.email, tc.sort)
	}
	mustExec(t, pool, insertGroup, c, k1, strings.Repeat("é", 60), nil, 2)

	mustFail(t, pool, "topic in use deleted", "delete from topics where id = $1", k1)
	mustFail(t, pool, "class with groups deleted", "delete from classes where id = $1", c)
}

func TestProgressDownAndUp(t *testing.T) {
	pool := migrated(t)
	y := activeYear(t, pool)
	c := newClass(t, pool, y, "9A")
	mustExec(t, pool, insertGroup, c, topicID(t, pool, "K2"), "Kelompok 1", nil, 0)
	newMaterial(t, pool, "tetap ada")

	goose(t, "down-to", "7")
	for _, name := range progressTables {
		if tableExists(t, pool, name) {
			t.Errorf("table %s still exists after down-to 7", name)
		}
	}
	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from materials").Scan(&n); err != nil || n != 1 {
		t.Fatalf("materials after rollback = %d (%v); Fase 2 data must be untouched", n, err)
	}
	goose(t, "up")
	if err := pool.QueryRow(ctx(t), "select count(*) from academic_years").Scan(&n); err != nil || n != 1 {
		t.Fatalf("academic years after re-up = %d (%v), want the single seed", n, err)
	}
}
