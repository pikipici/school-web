// Package users reads and updates admin accounts.
package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// ErrNotFound means no account has that username.
var ErrNotFound = errors.New("users: not found")

// ErrAdminExists means an admin account already exists; the site allows only one.
var ErrAdminExists = errors.New("users: admin already exists")

var usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

// ValidUsername reports whether s is an acceptable (already lower-case) username.
func ValidUsername(s string) bool { return usernamePattern.MatchString(s) }

// User is the minimum an authentication check needs.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

// Store is the data access layer for admin accounts.
type Store struct {
	q   *dbsql.Queries
	now func() time.Time
}

// New returns a Store. now may be nil (defaults to time.Now).
func New(db dbsql.DBTX, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{q: dbsql.New(db), now: now}
}

// FindByUsername looks up an account by its (case-insensitive) username.
func (s *Store) FindByUsername(ctx context.Context, username string) (User, error) {
	row, err := s.q.GetAdminUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("users: find: %w", err)
	}
	return User{ID: row.ID, Username: row.Username, PasswordHash: row.PasswordHash}, nil
}

// CreateFirstAdmin creates the only admin account. It returns ErrAdminExists
// if any admin is already present; the check and insert are one statement.
func (s *Store) CreateFirstAdmin(ctx context.Context, username, hash string) (int64, error) {
	id, err := s.q.CreateFirstAdminUser(ctx, dbsql.CreateFirstAdminUserParams{Username: username, PasswordHash: hash})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAdminExists
	}
	if err != nil {
		return 0, fmt.Errorf("users: create first admin: %w", err)
	}
	return id, nil
}

// MarkLogin records a successful login time.
func (s *Store) MarkLogin(ctx context.Context, id int64) error {
	err := s.q.TouchAdminUserLogin(ctx, dbsql.TouchAdminUserLoginParams{ID: id, LastLoginAt: ts(s.now())})
	if err != nil {
		return fmt.Errorf("users: mark login: %w", err)
	}
	return nil
}

// SetPasswordHash replaces the stored hash, e.g. after a parameter upgrade.
func (s *Store) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	err := s.q.UpdateAdminUserPasswordHash(ctx, dbsql.UpdateAdminUserPasswordHashParams{
		ID: id, PasswordHash: hash, UpdatedAt: ts(s.now()),
	})
	if err != nil {
		return fmt.Errorf("users: set password hash: %w", err)
	}
	return nil
}

func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}
