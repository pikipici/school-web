// Package users reads and updates admin accounts.
package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// ErrNotFound means no account has that username.
var ErrNotFound = errors.New("users: not found")

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
