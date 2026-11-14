package progress_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/progress"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func setup(t *testing.T) (*progress.Store, *pgxpool.Pool, *clock) {
	t.Helper()
	pool := testdb.OpenMigrated(t)
	clk := &clock{now: time.Date(2026, 11, 15, 1, 0, 0, 0, time.UTC)}
	return progress.New(pool, clk.Now), pool, clk
}

func topic(t *testing.T, pool *pgxpool.Pool, code string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), "select id from topics where code=$1", code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func activeYear(t *testing.T, s *progress.Store) progress.Year {
	t.Helper()
	ys, err := s.Years(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, y := range ys {
		if y.Active {
			return y
		}
	}
	t.Fatal("no active year")
	return progress.Year{}
}

func mustClass(t *testing.T, s *progress.Store, year int64, code string) progress.Class {
	t.Helper()
	c, err := s.CreateClass(context.Background(), progress.ClassInput{YearID: year, Code: code})
	if err != nil {
		t.Fatalf("create class %q: %v", code, err)
	}
	return c
}

func mustGroup(t *testing.T, s *progress.Store, class, topicID int64, name, email string, order int32) progress.Group {
	t.Helper()
	g, err := s.CreateGroup(context.Background(), progress.GroupInput{ClassID: class, TopicID: topicID, Name: name, Email: email, SortOrder: order})
	if err != nil {
		t.Fatalf("create group %q: %v", name, err)
	}
	return g
}

func TestValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		label string
		want  string
		ok    bool
	}{
		{"2027/2028", "2027/2028", true},
		{" 2027/2028 ", "2027/2028", true},
		{"2027/2029", "", false},
		{"2028/2027", "", false},
		{"2027-2028", "", false},
		{"27/28", "", false},
		{"", "", false},
	} {
		got, err := progress.ValidateYearLabel(tc.label)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ValidateYearLabel(%q) = %q, %v", tc.label, got, err)
		}
		if err != nil && !errors.Is(err, progress.ErrInvalid) {
			t.Errorf("ValidateYearLabel(%q) error %v is not ErrInvalid", tc.label, err)
		}
	}

	for _, tc := range []struct {
		in progress.ClassInput
		ok bool
	}{
		{progress.ClassInput{YearID: 1, Code: " 9A ", Name: " Kelas 9A "}, true},
		{progress.ClassInput{YearID: 1, Code: "IX-B.2"}, true},
		{progress.ClassInput{YearID: 1, Code: strings.Repeat("A", 20)}, true},
		{progress.ClassInput{YearID: 1, Code: strings.Repeat("A", 21)}, false},
		{progress.ClassInput{YearID: 1, Code: "  "}, false},
		{progress.ClassInput{YearID: 1, Code: "-9A"}, false},
		{progress.ClassInput{YearID: 1, Code: "9/A"}, false},
		{progress.ClassInput{YearID: 1, Code: "9A", Name: strings.Repeat("é", 81)}, false},
		{progress.ClassInput{YearID: 1, Code: "9A", Name: "a\u0007b"}, false},
		{progress.ClassInput{YearID: 0, Code: "9A"}, false},
	} {
		got, err := progress.ValidateClass(tc.in)
		if (err == nil) != tc.ok {
			t.Errorf("ValidateClass(%+v) err = %v", tc.in, err)
		}
		if err == nil && (got.Code != strings.TrimSpace(tc.in.Code) || got.Name != strings.TrimSpace(tc.in.Name)) {
			t.Errorf("ValidateClass(%+v) did not trim: %+v", tc.in, got)
		}
	}

	for _, tc := range []struct {
		in        progress.GroupInput
		wantEmail string
		ok        bool
	}{
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: " Kelompok 1 ", Email: " Kelompok1.9A.2627.Informatika@Gmail.com "}, "kelompok1.9a.2627.informatika@gmail.com", true},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: strings.Repeat("界", 60)}, "", true},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: strings.Repeat("界", 61)}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: " "}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: "Kel\nompok"}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 0, Name: "K"}, "", false},
		{progress.GroupInput{ClassID: 0, TopicID: 1, Name: "K"}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: "K", Email: "bukan email"}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: "K", Email: "a@b"}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: "K", Email: strings.Repeat("a", 245) + "@gmail.com"}, "", false},
		{progress.GroupInput{ClassID: 1, TopicID: 1, Name: "K", SortOrder: -1}, "", false},
	} {
		got, err := progress.ValidateGroup(tc.in)
		if (err == nil) != tc.ok {
			t.Errorf("ValidateGroup(%+v) err = %v", tc.in, err)
			continue
		}
		if err == nil && got.Email != tc.wantEmail {
			t.Errorf("ValidateGroup email = %q, want %q", got.Email, tc.wantEmail)
		}
	}
}

func TestYears(t *testing.T) {
	s, _, clk := setup(t)
	ctx := context.Background()
	first := activeYear(t, s)
	if first.Label != "2026/2027" {
		t.Fatalf("seed year = %q", first.Label)
	}
	next, err := s.CreateYear(ctx, " 2027/2028 ")
	if err != nil || next.Label != "2027/2028" || next.Active {
		t.Fatalf("CreateYear = %+v, %v", next, err)
	}
	if _, err := s.CreateYear(ctx, "2027/2028"); !errors.Is(err, progress.ErrDuplicate) {
		t.Fatalf("duplicate year err = %v", err)
	}
	if _, err := s.CreateYear(ctx, "2027/2029"); !errors.Is(err, progress.ErrInvalid) {
		t.Fatalf("bad year err = %v", err)
	}

	clk.Advance(time.Minute)
	if err := s.ActivateYear(ctx, next.ID); err != nil {
		t.Fatal(err)
	}
	ys, err := s.Years(ctx)
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, y := range ys {
		if y.Active {
			active++
			if y.ID != next.ID {
				t.Errorf("wrong active year %q", y.Label)
			}
		}
	}
	if active != 1 || len(ys) != 2 || ys[0].Label != "2027/2028" {
		t.Fatalf("years after switch = %+v", ys)
	}
	if err := s.ActivateYear(ctx, 999999); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("activate missing err = %v", err)
	}
	if got := activeYear(t, s); got.ID != next.ID {
		t.Fatal("failed activation changed the active year")
	}
}

func TestClassLifecycle(t *testing.T) {
	s, pool, clk := setup(t)
	ctx := context.Background()
	y := activeYear(t, s)

	c, err := s.CreateClass(ctx, progress.ClassInput{YearID: y.ID, Code: " 9A ", Name: " Kelas pagi "})
	if err != nil {
		t.Fatal(err)
	}
	if c.Code != "9A" || c.Name != "Kelas pagi" || !c.Active || c.YearID != y.ID || !c.CreatedAt.Equal(clk.now) {
		t.Fatalf("created class = %+v", c)
	}
	if _, err := s.CreateClass(ctx, progress.ClassInput{YearID: y.ID, Code: "9a"}); !errors.Is(err, progress.ErrDuplicate) {
		t.Fatalf("duplicate class err = %v", err)
	}
	if _, err := s.CreateClass(ctx, progress.ClassInput{YearID: 999999, Code: "9B"}); !errors.Is(err, progress.ErrInvalidReference) {
		t.Fatalf("unknown year err = %v", err)
	}
	b := mustClass(t, s, y.ID, "9B")

	clk.Advance(time.Second)
	upd, err := s.UpdateClass(ctx, c.ID, c.UpdatedAt, progress.ClassInput{YearID: y.ID, Code: "9A", Name: "", Active: false})
	if err != nil || upd.Active || upd.Name != "" || !upd.UpdatedAt.After(c.UpdatedAt) {
		t.Fatalf("UpdateClass = %+v, %v", upd, err)
	}
	if _, err := s.UpdateClass(ctx, c.ID, c.UpdatedAt, progress.ClassInput{YearID: y.ID, Code: "9A"}); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("stale update err = %v", err)
	}
	if _, err := s.UpdateClass(ctx, 999999, c.UpdatedAt, progress.ClassInput{YearID: y.ID, Code: "9Z"}); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("missing update err = %v", err)
	}
	if _, err := s.UpdateClass(ctx, b.ID, b.UpdatedAt, progress.ClassInput{YearID: y.ID, Code: "9A"}); !errors.Is(err, progress.ErrDuplicate) {
		t.Fatalf("rename onto existing err = %v", err)
	}

	mustGroup(t, s, b.ID, topic(t, pool, "K1"), "Kelompok 1", "", 0)
	list, err := s.Classes(ctx, y.ID)
	if err != nil || len(list) != 2 || list[0].Code != "9A" || list[1].GroupCount != 1 || list[0].GroupCount != 0 {
		t.Fatalf("Classes = %+v, %v", list, err)
	}
	if _, err := s.DeleteClass(ctx, b.ID); !errors.Is(err, progress.ErrInUse) {
		t.Fatalf("delete class with groups err = %v", err)
	}
	code, err := s.DeleteClass(ctx, c.ID)
	if err != nil || code != "9A" {
		t.Fatalf("DeleteClass = %q, %v", code, err)
	}
	if _, err := s.DeleteClass(ctx, c.ID); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("second delete err = %v", err)
	}
	if _, err := s.Class(ctx, c.ID); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("get deleted err = %v", err)
	}
}

func TestGroupLifecycle(t *testing.T) {
	s, pool, clk := setup(t)
	ctx := context.Background()
	y := activeYear(t, s)
	a := mustClass(t, s, y.ID, "9A")
	b := mustClass(t, s, y.ID, "9B")
	k1, k2 := topic(t, pool, "K1"), topic(t, pool, "K2")

	g, err := s.CreateGroup(ctx, progress.GroupInput{ClassID: a.ID, TopicID: k1, Name: " Kelompok 1 ", Email: " Kelompok1.9A.2627.Informatika@gmail.com ", SortOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Kelompok 1" || g.Email == nil || *g.Email != "kelompok1.9a.2627.informatika@gmail.com" || !g.Active || g.SortOrder != 1 {
		t.Fatalf("created group = %+v", g)
	}
	noMail := mustGroup(t, s, a.ID, k1, "Kelompok 2", "", 0)
	if noMail.Email != nil {
		t.Fatalf("empty email stored as %q", *noMail.Email)
	}
	if _, err := s.CreateGroup(ctx, progress.GroupInput{ClassID: a.ID, TopicID: k1, Name: "kelompok 1"}); !errors.Is(err, progress.ErrDuplicate) {
		t.Fatalf("duplicate group err = %v", err)
	}
	for name, in := range map[string]progress.GroupInput{
		"unknown topic": {ClassID: a.ID, TopicID: 999999, Name: "X"},
		"unknown class": {ClassID: 999999, TopicID: k1, Name: "X"},
	} {
		if _, err := s.CreateGroup(ctx, in); !errors.Is(err, progress.ErrInvalidReference) {
			t.Errorf("%s err = %v", name, err)
		}
	}

	list, err := s.Groups(ctx, a.ID)
	if err != nil || len(list) != 2 || list[0].Name != "Kelompok 2" || list[1].Name != "Kelompok 1" {
		t.Fatalf("Groups order = %+v, %v (want sort_order first)", list, err)
	}

	clk.Advance(time.Second)
	moved, err := s.UpdateGroup(ctx, g.ID, g.UpdatedAt, progress.GroupInput{ClassID: b.ID, TopicID: k2, Name: "Kelompok 1", Email: "", SortOrder: 0, Active: false})
	if err != nil || moved.ClassID != b.ID || moved.TopicID != k2 || moved.Email != nil || moved.Active {
		t.Fatalf("UpdateGroup = %+v, %v", moved, err)
	}
	if _, err := s.UpdateGroup(ctx, g.ID, g.UpdatedAt, progress.GroupInput{ClassID: b.ID, TopicID: k2, Name: "Z"}); !errors.Is(err, progress.ErrConflict) {
		t.Fatalf("stale group update err = %v", err)
	}
	if _, err := s.UpdateGroup(ctx, 999999, g.UpdatedAt, progress.GroupInput{ClassID: b.ID, TopicID: k2, Name: "Z"}); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("missing group update err = %v", err)
	}
	got, err := s.Group(ctx, g.ID)
	if err != nil || got.ClassID != b.ID {
		t.Fatalf("Group = %+v, %v", got, err)
	}
	name, err := s.DeleteGroup(ctx, g.ID)
	if err != nil || name != "Kelompok 1" {
		t.Fatalf("DeleteGroup = %q, %v", name, err)
	}
	if _, err := s.DeleteGroup(ctx, g.ID); !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("second group delete err = %v", err)
	}
	if _, err := pool.Exec(ctx, "delete from topics where id=$1", k1); err == nil {
		t.Fatal("topic in use was deleted")
	}
}

func TestPublicProgressShowsOnlyActiveSafeRows(t *testing.T) {
	s, pool, _ := setup(t)
	ctx := context.Background()
	y := activeYear(t, s)
	k1, k3 := topic(t, pool, "K1"), topic(t, pool, "K3")

	a := mustClass(t, s, y.ID, "9A")
	b := mustClass(t, s, y.ID, "9b")
	hidden := mustClass(t, s, y.ID, "9C")
	mustGroup(t, s, a.ID, k1, "Kelompok 2", "rahasia2@gmail.com", 2)
	mustGroup(t, s, a.ID, k3, "Kelompok 1", "rahasia1@gmail.com", 1)
	off := mustGroup(t, s, a.ID, k1, "Kelompok Nonaktif", "", 3)
	mustGroup(t, s, b.ID, k1, "Kelompok 100%", "", 0)
	mustGroup(t, s, hidden.ID, k1, "Kelompok Tersembunyi", "", 0)
	if _, err := s.UpdateGroup(ctx, off.ID, off.UpdatedAt, progress.GroupInput{ClassID: a.ID, TopicID: k1, Name: off.Name, SortOrder: 3, Active: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateClass(ctx, hidden.ID, hidden.UpdatedAt, progress.ClassInput{YearID: y.ID, Code: "9C", Active: false}); err != nil {
		t.Fatal(err)
	}
	// A group in another, inactive year never shows.
	old, err := s.CreateYear(ctx, "2025/2026")
	if err != nil {
		t.Fatal(err)
	}
	oc := mustClass(t, s, old.ID, "9A")
	mustGroup(t, s, oc.ID, k1, "Kelompok Lama", "", 0)

	p, err := s.PublicProgress(ctx, progress.PublicFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Year != "2026/2027" {
		t.Errorf("year = %q", p.Year)
	}
	var classes []string
	for _, c := range p.Classes {
		classes = append(classes, c.Code)
	}
	if strings.Join(classes, ",") != "9A,9b" {
		t.Errorf("public classes = %v", classes)
	}
	var names []string
	for _, g := range p.Groups {
		names = append(names, g.ClassCode+"/"+g.Name)
	}
	if strings.Join(names, ",") != "9A/Kelompok 1,9A/Kelompok 2,9b/Kelompok 100%" {
		t.Fatalf("public groups = %v", names)
	}
	first := p.Groups[0]
	if first.Topic.Code != "K3" || first.Topic.Title == "" || first.Submission != progress.SubmissionNone ||
		first.Review != progress.ReviewNotChecked || first.Notebook != progress.NotebookNotReceived || first.Complete || first.UpdatedAt.IsZero() {
		t.Errorf("first public row = %+v", first)
	}

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "rahasia") || strings.Contains(strings.ToLower(string(raw)), "email") {
		t.Fatalf("public payload leaks email: %s", raw)
	}
	var rows struct {
		Groups []map[string]any `json:"groups"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range rows.Groups[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "classCode,complete,name,notebook,review,submission,topic,updatedAt" {
		t.Errorf("public row keys = %v", keys)
	}

	for _, tc := range []struct {
		f    progress.PublicFilter
		want int
	}{
		{progress.PublicFilter{Class: "9B"}, 1},
		{progress.PublicFilter{Class: "9a"}, 2},
		{progress.PublicFilter{Class: "9C"}, 0},
		{progress.PublicFilter{Query: "kelompok 1"}, 2},
		{progress.PublicFilter{Query: "100%"}, 1},
		{progress.PublicFilter{Query: "%"}, 1},
		{progress.PublicFilter{Query: "_"}, 0},
		{progress.PublicFilter{Class: "9A", Query: "2"}, 1},
	} {
		got, err := s.PublicProgress(ctx, tc.f)
		if err != nil || len(got.Groups) != tc.want {
			t.Errorf("filter %+v = %d rows, %v; want %d", tc.f, len(got.Groups), err, tc.want)
		}
	}
	for _, f := range []progress.PublicFilter{{Query: strings.Repeat("a", 61)}, {Class: strings.Repeat("A", 21)}, {Class: "9/A"}} {
		if _, err := s.PublicProgress(ctx, f); !errors.Is(err, progress.ErrInvalid) {
			t.Errorf("filter %+v err = %v, want ErrInvalid", f, err)
		}
	}

	sum, err := s.PublicSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sum != (progress.Summary{Total: 3, None: 3}) {
		t.Errorf("summary = %+v", sum)
	}
}

func TestPublicProgressEmpty(t *testing.T) {
	s, _, _ := setup(t)
	p, err := s.PublicProgress(context.Background(), progress.PublicFilter{})
	if err != nil || p.Year != "2026/2027" || len(p.Groups) != 0 || len(p.Classes) != 0 || p.Groups == nil || p.Classes == nil {
		t.Fatalf("empty progress = %+v, %v (slices must be non-nil for JSON [])", p, err)
	}
	sum, err := s.PublicSummary(context.Background())
	if err != nil || sum != (progress.Summary{}) {
		t.Fatalf("empty summary = %+v, %v", sum, err)
	}
}
