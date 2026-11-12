// Package testdb provides an isolated PostgreSQL database for integration tests.
//
// Tests read TEST_DATABASE_URL. The database name must end in "_test" so a
// misconfigured environment can never reset the production database.
package testdb

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	envURL     = "TEST_DATABASE_URL"
	envRequire = "REQUIRE_TEST_DB"
	suffix     = "_test"
)

// ValidateTestDSN reports whether dsn points to a database reserved for tests.
// Errors never include the DSN, so credentials cannot leak into test output.
func ValidateTestDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("test database URL is empty")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Host == "" {
		return errors.New("test database URL is not a valid URL")
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return errors.New("test database URL has no database name")
	}
	if !strings.HasSuffix(name, suffix) {
		return errors.New("test database name must end in " + suffix)
	}
	return nil
}

// Open connects to the test database and closes the pool when the test ends.
// Without TEST_DATABASE_URL the test is skipped, unless REQUIRE_TEST_DB=1, in
// which case it fails so a CI gate cannot pass silently without a database.
func Open(t testing.TB) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(envURL)
	if dsn == "" {
		if os.Getenv(envRequire) == "1" {
			t.Fatalf("%s is required but not set", envURL)
		}
		t.Skipf("%s not set; skipping integration test", envURL)
	}
	if err := ValidateTestDSN(dsn); err != nil {
		t.Fatalf("refusing to use %s: %v", envURL, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("open test database pool failed")
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal("ping test database failed")
	}
	return pool
}

// Reset drops and recreates the public schema so each test starts empty.
func Reset(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, stmt := range []string{
		"drop schema if exists public cascade",
		"create schema public",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset test database: %v", err)
		}
	}
}
