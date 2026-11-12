package testdb_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

func TestValidateTestDSN(t *testing.T) {
	cases := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{"test database accepted", "postgres://u:p@127.0.0.1:5432/informatika_monitoring_test?sslmode=disable", false},
		{"production database rejected", "postgres://u:p@127.0.0.1:5432/informatika_monitoring?sslmode=disable", true},
		{"empty rejected", "", true},
		{"malformed rejected", "not a dsn", true},
		{"missing database name rejected", "postgres://u:p@127.0.0.1:5432/", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := testdb.ValidateTestDSN(tc.dsn)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateTestDSN(%q) error = %v, wantErr %v", tc.dsn, err, tc.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), "p@") {
				t.Fatalf("error leaks credentials: %v", err)
			}
		})
	}
}

func TestOpenConnectsToIsolatedTestDatabase(t *testing.T) {
	pool := testdb.Open(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var name string
	if err := pool.QueryRow(ctx, "select current_database()").Scan(&name); err != nil {
		t.Fatalf("query current_database: %v", err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("connected to %q, want a database ending in _test", name)
	}
}

func TestResetLeavesEmptyPublicSchema(t *testing.T) {
	pool := testdb.Open(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "create table leftover (id int)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	testdb.Reset(t, pool)

	var count int
	if err := pool.QueryRow(ctx, "select count(*) from information_schema.tables where table_schema = 'public'").Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count != 0 {
		t.Fatalf("public schema has %d tables after Reset, want 0", count)
	}
}
