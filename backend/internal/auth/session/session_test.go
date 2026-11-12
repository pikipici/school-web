package session_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

// clock is a controllable time source so expiry rules are tested without sleeping.
type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type env struct {
	pool *pgxpool.Pool
	svc  *session.Service
	clk  *clock
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testdb.OpenMigrated(t)
	clk := &clock{now: time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)}
	svc := session.New(pool, session.Options{Now: clk.Now})
	return &env{pool: pool, svc: svc, clk: clk}
}

func (e *env) ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return c
}

func (e *env) user(t *testing.T, name string) int64 {
	t.Helper()
	var id int64
	err := e.pool.QueryRow(e.ctx(t),
		"insert into admin_users (username, password_hash) values ($1, 'x') returning id", name).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func (e *env) create(t *testing.T, uid int64) session.Issued {
	t.Helper()
	iss, err := e.svc.Create(e.ctx(t), uid, "203.0.113.7", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return iss
}

func (e *env) count(t *testing.T) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(e.ctx(t), "select count(*) from admin_sessions").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateIssuesRandomTokensAndStoresOnlyTheHash(t *testing.T) {
	e := setup(t)
	uid := e.user(t, "fikri")

	a := e.create(t, uid)
	b := e.create(t, uid)

	if len(a.Token) != 43 || len(a.CSRFToken) != 43 {
		t.Fatalf("token lengths = %d, %d; want 43 (32 bytes base64url)", len(a.Token), len(a.CSRFToken))
	}
	if a.Token == b.Token || a.CSRFToken == b.CSRFToken {
		t.Fatal("two sessions share a token")
	}
	if a.Token == a.CSRFToken {
		t.Fatal("session token and CSRF token must differ")
	}
	if want := e.clk.Now().Add(12 * time.Hour); !a.ExpiresAt.Equal(want) {
		t.Fatalf("ExpiresAt = %v; want %v", a.ExpiresAt, want)
	}

	var stored []byte
	if err := e.pool.QueryRow(e.ctx(t),
		"select token_hash from admin_sessions order by id limit 1").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(a.Token))
	if string(stored) != string(sum[:]) {
		t.Fatal("stored token_hash is not SHA-256 of the issued token")
	}

	var leaks int
	if err := e.pool.QueryRow(e.ctx(t),
		"select count(*) from admin_sessions s where s::text like '%' || $1 || '%'", a.Token).Scan(&leaks); err != nil {
		t.Fatal(err)
	}
	if leaks != 0 {
		t.Fatal("raw session token appears in the database row")
	}
}

func TestCreateForUnknownUserFails(t *testing.T) {
	e := setup(t)
	if _, err := e.svc.Create(e.ctx(t), 999999, "", ""); err == nil {
		t.Fatal("Create for a nonexistent user succeeded")
	}
}

func TestValidateReturnsSession(t *testing.T) {
	e := setup(t)
	uid := e.user(t, "fikri")
	iss := e.create(t, uid)

	e.clk.Advance(time.Minute)
	got, err := e.svc.Validate(e.ctx(t), iss.Token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got.UserID != uid || got.Username != "fikri" || got.CSRFToken != iss.CSRFToken {
		t.Fatalf("Validate = %+v; want user %d/fikri with the issued CSRF token", got, uid)
	}
}

func TestValidateRejectsUnknownAndMalformedTokens(t *testing.T) {
	e := setup(t)
	e.create(t, e.user(t, "fikri"))

	for name, tok := range map[string]string{
		"empty":         "",
		"too short":     "abc",
		"unknown valid": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"bad alphabet":  "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!",
		"too long":      "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := e.svc.Validate(e.ctx(t), tok); !errors.Is(err, session.ErrInvalid) {
				t.Fatalf("Validate(%q) = %v; want ErrInvalid", name, err)
			}
		})
	}
}

func TestIdleTimeoutExpiresSessionAndDeletesRow(t *testing.T) {
	e := setup(t)
	iss := e.create(t, e.user(t, "fikri"))

	e.clk.Advance(30*time.Minute + time.Second)
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("Validate after idle timeout = %v; want ErrInvalid", err)
	}
	if n := e.count(t); n != 0 {
		t.Fatalf("expired session row not deleted; %d remain", n)
	}
}

func TestActivityKeepsSessionAliveUntilAbsoluteLimit(t *testing.T) {
	e := setup(t)
	iss := e.create(t, e.user(t, "fikri"))

	// Active every 20 minutes: never idle for 30, so it must stay valid.
	for elapsed := 20 * time.Minute; elapsed <= 11*time.Hour+40*time.Minute; elapsed += 20 * time.Minute {
		e.clk.Advance(20 * time.Minute)
		if _, err := e.svc.Validate(e.ctx(t), iss.Token); err != nil {
			t.Fatalf("Validate at +%v = %v; want valid", elapsed, err)
		}
	}

	// Still "active", but past the 12 hour absolute limit.
	e.clk.Advance(21 * time.Minute)
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("Validate past absolute limit = %v; want ErrInvalid", err)
	}
}

func TestLastSeenUpdateIsThrottled(t *testing.T) {
	e := setup(t)
	iss := e.create(t, e.user(t, "fikri"))

	lastSeen := func() time.Time {
		var ts time.Time
		if err := e.pool.QueryRow(e.ctx(t), "select last_seen_at from admin_sessions").Scan(&ts); err != nil {
			t.Fatal(err)
		}
		return ts.UTC()
	}
	start := lastSeen()

	e.clk.Advance(10 * time.Second)
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); err != nil {
		t.Fatal(err)
	}
	if !lastSeen().Equal(start) {
		t.Fatal("last_seen_at written on every request; writes must be throttled")
	}

	e.clk.Advance(2 * time.Minute)
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); err != nil {
		t.Fatal(err)
	}
	if !lastSeen().Equal(e.clk.Now()) {
		t.Fatalf("last_seen_at = %v; want %v after throttle window", lastSeen(), e.clk.Now())
	}
}

func TestRevoke(t *testing.T) {
	e := setup(t)
	iss := e.create(t, e.user(t, "fikri"))

	if err := e.svc.Revoke(e.ctx(t), iss.Token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("Validate after Revoke = %v; want ErrInvalid", err)
	}
	if err := e.svc.Revoke(e.ctx(t), iss.Token); err != nil {
		t.Fatalf("second Revoke must be idempotent: %v", err)
	}
	if err := e.svc.Revoke(e.ctx(t), "not-a-token"); err != nil {
		t.Fatalf("Revoke of malformed token must not error: %v", err)
	}
}

func TestRevokeAllForUserLeavesOtherUsers(t *testing.T) {
	e := setup(t)
	fikri := e.user(t, "fikri")
	other := e.user(t, "guru2")

	a, b := e.create(t, fikri), e.create(t, fikri)
	keep := e.create(t, other)

	if err := e.svc.RevokeAllForUser(e.ctx(t), fikri); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	for _, tok := range []string{a.Token, b.Token} {
		if _, err := e.svc.Validate(e.ctx(t), tok); !errors.Is(err, session.ErrInvalid) {
			t.Fatalf("revoked session still valid: %v", err)
		}
	}
	if _, err := e.svc.Validate(e.ctx(t), keep.Token); err != nil {
		t.Fatalf("other user's session was revoked: %v", err)
	}
}

func TestPurgeExpiredRemovesIdleAndAbsoluteExpired(t *testing.T) {
	e := setup(t)
	uid := e.user(t, "fikri")

	idle := e.create(t, uid)
	e.clk.Advance(20 * time.Minute)
	active := e.create(t, uid)
	e.clk.Advance(11 * time.Minute) // idle session now 31 min old, active one 11

	n, err := e.svc.PurgeExpired(e.ctx(t))
	if err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if n != 1 {
		t.Fatalf("PurgeExpired removed %d rows; want 1", n)
	}
	if _, err := e.svc.Validate(e.ctx(t), idle.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("idle session survived purge: %v", err)
	}
	if _, err := e.svc.Validate(e.ctx(t), active.Token); err != nil {
		t.Fatalf("active session removed by purge: %v", err)
	}
}

func TestDeletingUserInvalidatesSession(t *testing.T) {
	e := setup(t)
	uid := e.user(t, "fikri")
	iss := e.create(t, uid)

	if _, err := e.pool.Exec(e.ctx(t), "delete from admin_users where id = $1", uid); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Validate(e.ctx(t), iss.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("Validate after user delete = %v; want ErrInvalid", err)
	}
}
