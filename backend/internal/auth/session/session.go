// Package session manages server-side admin sessions stored in PostgreSQL.
//
// The browser holds a random token; the database stores only its SHA-256 hash,
// so a leaked database dump cannot be replayed as a session cookie.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

const (
	DefaultIdleTimeout     = 30 * time.Minute
	DefaultAbsoluteTimeout = 12 * time.Hour

	// touchInterval limits last_seen_at writes to at most one per minute per session.
	touchInterval = time.Minute

	tokenBytes   = 32
	tokenLength  = 43 // base64url (no padding) of 32 bytes
	maxUserAgent = 512
)

// ErrInvalid is returned for every reason a session cannot be used (unknown,
// malformed, expired, idle, revoked) so callers cannot leak which one applied.
var ErrInvalid = errors.New("session: invalid")

// Options configures a Service. Zero values use the defaults.
type Options struct {
	IdleTimeout     time.Duration
	AbsoluteTimeout time.Duration
	// Now overrides the clock, for tests.
	Now func() time.Time
}

// Service creates, validates, and revokes admin sessions.
type Service struct {
	q        *dbsql.Queries
	idle     time.Duration
	absolute time.Duration
	now      func() time.Time
}

// Issued is returned once at login; Token must only be sent to the browser as
// an HttpOnly cookie and never stored or logged.
type Issued struct {
	Token     string
	CSRFToken string
	ExpiresAt time.Time
}

// Session is the validated identity attached to a request.
type Session struct {
	ID        int64
	UserID    int64
	Username  string
	CSRFToken string
	ExpiresAt time.Time
}

// New returns a Service backed by db.
func New(db dbsql.DBTX, opts Options) *Service {
	s := &Service{
		q:        dbsql.New(db),
		idle:     opts.IdleTimeout,
		absolute: opts.AbsoluteTimeout,
		now:      opts.Now,
	}
	if s.idle <= 0 {
		s.idle = DefaultIdleTimeout
	}
	if s.absolute <= 0 {
		s.absolute = DefaultAbsoluteTimeout
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

// Create starts a session for userID. ip and userAgent are recorded for audit
// when valid; they never affect authorization.
func (s *Service) Create(ctx context.Context, userID int64, ip, userAgent string) (Issued, error) {
	token, err := randomToken()
	if err != nil {
		return Issued{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return Issued{}, err
	}

	now := s.now().UTC()
	expires := now.Add(s.absolute)

	if addr, err := netip.ParseAddr(ip); err == nil {
		ip = addr.String()
	} else {
		ip = ""
	}
	if len(userAgent) > maxUserAgent {
		userAgent = userAgent[:maxUserAgent]
	}

	_, err = s.q.CreateAdminSession(ctx, dbsql.CreateAdminSessionParams{
		UserID:    userID,
		TokenHash: hashToken(token),
		CsrfToken: csrf,
		CreatedAt: ts(now),
		ExpiresAt: ts(expires),
		Ip:        ip,
		UserAgent: userAgent,
	})
	if err != nil {
		return Issued{}, fmt.Errorf("session: create: %w", err)
	}
	return Issued{Token: token, CSRFToken: csrf, ExpiresAt: expires}, nil
}

// Validate returns the session for token, or ErrInvalid. It enforces both the
// idle timeout (sliding) and the absolute lifetime, deletes dead sessions, and
// refreshes last_seen_at at most once per touchInterval.
func (s *Service) Validate(ctx context.Context, token string) (*Session, error) {
	if !wellFormed(token) {
		return nil, ErrInvalid
	}

	row, err := s.q.GetAdminSessionByTokenHash(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("session: lookup: %w", err)
	}

	now := s.now().UTC()
	if !now.Before(row.ExpiresAt.Time) || now.Sub(row.LastSeenAt.Time) > s.idle {
		_ = s.q.DeleteAdminSessionByID(ctx, row.ID)
		return nil, ErrInvalid
	}

	if now.Sub(row.LastSeenAt.Time) >= touchInterval {
		if err := s.q.TouchAdminSession(ctx, dbsql.TouchAdminSessionParams{ID: row.ID, LastSeenAt: ts(now)}); err != nil {
			return nil, fmt.Errorf("session: touch: %w", err)
		}
	}

	return &Session{
		ID:        row.ID,
		UserID:    row.UserID,
		Username:  row.Username,
		CSRFToken: row.CsrfToken,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

// Revoke deletes the session for token. It is idempotent and ignores
// malformed tokens, so logout always succeeds.
func (s *Service) Revoke(ctx context.Context, token string) error {
	if !wellFormed(token) {
		return nil
	}
	if err := s.q.DeleteAdminSessionByTokenHash(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("session: revoke: %w", err)
	}
	return nil
}

// RevokeAllForUser ends every session of a user, e.g. after a password reset.
func (s *Service) RevokeAllForUser(ctx context.Context, userID int64) error {
	if err := s.q.DeleteAdminSessionsByUser(ctx, userID); err != nil {
		return fmt.Errorf("session: revoke all: %w", err)
	}
	return nil
}

// PurgeExpired removes sessions past their absolute or idle limit and returns
// how many were deleted.
func (s *Service) PurgeExpired(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	n, err := s.q.DeleteStaleAdminSessions(ctx, dbsql.DeleteStaleAdminSessionsParams{
		Now:        ts(now),
		IdleCutoff: ts(now.Add(-s.idle)),
	})
	if err != nil {
		return 0, fmt.Errorf("session: purge: %w", err)
	}
	return n, nil
}

func randomToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// wellFormed rejects obviously invalid tokens before touching the database.
func wellFormed(token string) bool {
	if len(token) != tokenLength {
		return false
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			return false
		}
	}
	return true
}
