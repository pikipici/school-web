package schema_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

const migrationsDir = "../../sql/migrations"

// goose runs the goose CLI against the test database. The DSN is passed via
// environment so it never appears in the process list or test output.
func goose(t *testing.T, args ...string) {
	t.Helper()

	bin, err := exec.LookPath("goose")
	if err != nil {
		if os.Getenv("REQUIRE_TEST_DB") == "1" {
			t.Fatal("goose binary is required but not found in PATH")
		}
		t.Skip("goose binary not found; skipping migration test")
	}

	dir, err := filepath.Abs(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"GOOSE_DRIVER=postgres",
		"GOOSE_DBSTRING="+os.Getenv("TEST_DATABASE_URL"),
		"GOOSE_MIGRATION_DIR="+dir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Output may echo statements but never the DSN (passed via env only).
		t.Fatalf("goose %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.Open(t)
	testdb.Reset(t, pool)
	goose(t, "up")
	return pool
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return c
}

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var ok bool
	err := pool.QueryRow(ctx(t),
		"select exists (select 1 from information_schema.tables where table_schema='public' and table_name=$1)", name).Scan(&ok)
	if err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return ok
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx(t), sql, args...); err != nil {
		t.Fatalf("exec failed: %v\nsql: %s", err, sql)
	}
}

func mustFail(t *testing.T, pool *pgxpool.Pool, why, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx(t), sql, args...); err == nil {
		t.Fatalf("expected failure (%s) but statement succeeded: %s", why, sql)
	}
}

func insertUser(t *testing.T, pool *pgxpool.Pool, username string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(ctx(t),
		"insert into admin_users (username, password_hash) values ($1, 'argon2id$placeholder') returning id", username).Scan(&id)
	if err != nil {
		t.Fatalf("insert user %q: %v", username, err)
	}
	return id
}

func hash32() []byte { return []byte(strings.Repeat("a", 32)) }

func TestUpCreatesTables(t *testing.T) {
	pool := migrated(t)
	for _, name := range []string{"admin_users", "admin_sessions", "audit_log"} {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing after goose up", name)
		}
	}
}

func TestAdminUsersConstraints(t *testing.T) {
	pool := migrated(t)

	insertUser(t, pool, "fikri")

	mustFail(t, pool, "duplicate username",
		"insert into admin_users (username, password_hash) values ('fikri', 'x')")
	mustFail(t, pool, "uppercase username",
		"insert into admin_users (username, password_hash) values ('Fikri2', 'x')")
	mustFail(t, pool, "username too short",
		"insert into admin_users (username, password_hash) values ('ab', 'x')")
	mustFail(t, pool, "empty password hash",
		"insert into admin_users (username, password_hash) values ('guru2', '')")
}

func TestAdminSessionsConstraintsAndCascade(t *testing.T) {
	pool := migrated(t)
	uid := insertUser(t, pool, "fikri")

	insert := `insert into admin_sessions (user_id, token_hash, csrf_token, expires_at)
	           values ($1, $2, 'csrf', now() + interval '1 hour')`
	mustExec(t, pool, insert, uid, hash32())

	mustFail(t, pool, "duplicate token hash", insert, uid, hash32())
	mustFail(t, pool, "token hash wrong length", insert, uid, []byte("short"))
	mustFail(t, pool, "unknown user",
		insert, int64(999999), []byte(strings.Repeat("b", 32)))
	mustFail(t, pool, "expires before created",
		`insert into admin_sessions (user_id, token_hash, csrf_token, created_at, expires_at)
		 values ($1, $2, 'csrf', now(), now() - interval '1 hour')`,
		uid, []byte(strings.Repeat("c", 32)))

	mustExec(t, pool, "delete from admin_users where id = $1", uid)
	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from admin_sessions").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("sessions remaining after user delete = %d, want 0 (cascade)", n)
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	pool := migrated(t)

	mustExec(t, pool,
		"insert into audit_log (action, outcome, detail) values ('auth.login', 'success', '{}'::jsonb)")

	mustFail(t, pool, "invalid outcome",
		"insert into audit_log (action, outcome) values ('auth.login', 'maybe')")
	mustFail(t, pool, "invalid action format",
		"insert into audit_log (action, outcome) values ('Bad Action!', 'success')")
	mustFail(t, pool, "update blocked", "update audit_log set outcome = 'failure'")
	mustFail(t, pool, "delete blocked", "delete from audit_log")

	// An audit trail must survive deleting the actor.
	uid := insertUser(t, pool, "fikri")
	mustExec(t, pool,
		"insert into audit_log (actor_user_id, action, outcome) values ($1, 'auth.logout', 'success')", uid)
	mustExec(t, pool, "delete from admin_users where id = $1", uid)

	var n int
	if err := pool.QueryRow(ctx(t), "select count(*) from audit_log").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("audit rows = %d, want 2 (history kept after user delete)", n)
	}
}

func TestDownThenUpRoundTrip(t *testing.T) {
	pool := migrated(t)

	goose(t, "down-to", "0")
	for _, name := range []string{"admin_users", "admin_sessions", "audit_log"} {
		if tableExists(t, pool, name) {
			t.Errorf("table %s still exists after down-to 0", name)
		}
	}

	goose(t, "up")
	for _, name := range []string{"admin_users", "admin_sessions", "audit_log"} {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing after re-up", name)
		}
	}
}
