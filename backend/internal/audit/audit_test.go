package audit_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

var fixedNow = time.Date(2026, 11, 13, 9, 30, 0, 0, time.UTC)

func setup(t *testing.T) (*pgxpool.Pool, *audit.Store) {
	t.Helper()
	pool := testdb.OpenMigrated(t)
	return pool, audit.New(pool, func() time.Time { return fixedNow })
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return c
}

type row struct {
	actor   *int64
	action  string
	outcome string
	ip      *string
	detail  map[string]string
	at      time.Time
}

func rows(t *testing.T, pool *pgxpool.Pool) []row {
	t.Helper()
	r, err := pool.Query(ctx(t),
		"select actor_user_id, action, outcome, host(ip), detail::text, occurred_at from audit_log order by id")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var out []row
	for r.Next() {
		var x row
		var detail string
		if err := r.Scan(&x.actor, &x.action, &x.outcome, &x.ip, &detail, &x.at); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(detail), &x.detail); err != nil {
			t.Fatalf("detail is not a JSON object of strings: %q", detail)
		}
		x.at = x.at.UTC()
		out = append(out, x)
	}
	return out
}

func TestRecordWritesAllFields(t *testing.T) {
	pool, store := setup(t)
	uid := int64(42)

	store.Record(ctx(t), audit.Event{
		Action: "auth.login", Outcome: "failure", UserID: &uid, IP: "203.0.113.7",
		Detail: map[string]string{"username": "fikri"},
	})

	got := rows(t, pool)
	if len(got) != 1 {
		t.Fatalf("rows = %d; want 1", len(got))
	}
	r := got[0]
	if r.actor == nil || *r.actor != 42 || r.action != "auth.login" || r.outcome != "failure" ||
		r.ip == nil || *r.ip != "203.0.113.7" || r.detail["username"] != "fikri" || !r.at.Equal(fixedNow) {
		t.Fatalf("unexpected row: %+v", r)
	}
}

func TestRecordStoresNullForMissingActorAndInvalidIP(t *testing.T) {
	pool, store := setup(t)

	store.Record(ctx(t), audit.Event{Action: "auth.logout", Outcome: "success"})
	store.Record(ctx(t), audit.Event{Action: "auth.logout", Outcome: "success", IP: "not-an-ip"})

	got := rows(t, pool)
	if len(got) != 2 {
		t.Fatalf("rows = %d; want 2 (a bad IP must not drop the event)", len(got))
	}
	for i, r := range got {
		if r.actor != nil || r.ip != nil {
			t.Fatalf("row %d: actor=%v ip=%v; want NULL for both", i, r.actor, r.ip)
		}
		if len(r.detail) != 0 {
			t.Fatalf("row %d: detail = %v; want empty object", i, r.detail)
		}
	}
}

func TestRecordRejectsMalformedEventsWithoutPanicking(t *testing.T) {
	pool, store := setup(t)

	for _, e := range []audit.Event{
		{Action: "", Outcome: "success"},
		{Action: "Bad Action!", Outcome: "success"},
		{Action: "auth.login", Outcome: "maybe"},
		{Action: strings.Repeat("a", 100), Outcome: "success"},
	} {
		store.Record(ctx(t), e)
	}
	if n := len(rows(t, pool)); n != 0 {
		t.Fatalf("rows = %d; malformed events must be dropped", n)
	}
}

func TestRecordDropsSensitiveDetailKeysAndBoundsSize(t *testing.T) {
	pool, store := setup(t)

	detail := map[string]string{
		"username":      "fikri",
		"password":      "kopi-hangat-di-pagi-hari",
		"Password_Hash": "$argon2id$v=19$...",
		"csrf_token":    "abc",
		"session_token": "abc",
		"Cookie":        "admin_session=abc",
		"secret":        "x",
		"long":          strings.Repeat("x", 1000),
	}
	store.Record(ctx(t), audit.Event{Action: "auth.login", Outcome: "failure", Detail: detail})

	got := rows(t, pool)
	if len(got) != 1 {
		t.Fatalf("rows = %d", len(got))
	}
	d := got[0].detail
	for _, k := range []string{"password", "Password_Hash", "csrf_token", "session_token", "Cookie", "secret"} {
		if _, ok := d[k]; ok {
			t.Errorf("sensitive key %q was stored", k)
		}
	}
	if d["username"] != "fikri" {
		t.Errorf("safe key dropped: %v", d)
	}
	if len(d["long"]) > 200 {
		t.Errorf("long value not truncated: %d bytes", len(d["long"]))
	}

	var dump string
	if err := pool.QueryRow(ctx(t), "select string_agg(detail::text, ' ') from audit_log").Scan(&dump); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dump, "kopi-hangat") || strings.Contains(dump, "argon2") {
		t.Fatal("secret material reached the audit table")
	}
}

func TestRecordCapsNumberOfDetailKeys(t *testing.T) {
	pool, store := setup(t)
	detail := map[string]string{}
	for i := 0; i < 50; i++ {
		detail["k"+strings.Repeat("a", i%5)+string(rune('a'+i%26))+string(rune('a'+i/26))] = "v"
	}
	store.Record(ctx(t), audit.Event{Action: "auth.login", Outcome: "success", Detail: detail})

	if n := len(rows(t, pool)[0].detail); n > 10 {
		t.Fatalf("stored %d detail keys; cap is 10", n)
	}
}

func TestRecordStillWritesAfterRequestContextIsCancelled(t *testing.T) {
	pool, store := setup(t)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel() // the client hung up, but the action happened and must be recorded
	store.Record(cancelled, audit.Event{Action: "auth.logout", Outcome: "success"})

	if n := len(rows(t, pool)); n != 1 {
		t.Fatalf("rows = %d; want 1", n)
	}
}

func TestRecordNeverPropagatesDatabaseFailure(t *testing.T) {
	// Nothing listens on port 1, so every query fails.
	pool, err := pgxpool.New(context.Background(), "postgres://nobody:nopass@127.0.0.1:1/nodb?connect_timeout=2")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := audit.New(pool, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		store.Record(context.Background(), audit.Event{Action: "auth.login", Outcome: "success"})
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Record hung on a failing database")
	}
}
