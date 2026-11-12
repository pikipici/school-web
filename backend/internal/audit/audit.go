// Package audit writes the append-only security audit trail.
//
// Recording an event never fails the caller: a login must not break because the
// audit insert did. Failures are logged without event contents. As defence in
// depth, Record also strips detail keys that look like secrets and bounds the
// size of what is stored, whatever the caller passes.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/netip"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

const (
	writeTimeout = 5 * time.Second

	maxDetailKeys   = 10
	maxDetailKeyLen = 64
	maxDetailValLen = 200
)

// actionPattern mirrors the audit_log_action_format database constraint.
var actionPattern = regexp.MustCompile(`^[a-z][a-z0-9_.]{1,63}$`)

// sensitiveKeyParts are substrings that mark a detail key as secret-bearing.
var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "cookie", "hash", "csrf",
	"authorization", "credential", "apikey", "api_key",
}

// Event describes one security-relevant action. Detail must not hold
// passwords, tokens, or free-form user input.
type Event struct {
	Action  string
	Outcome string // "success" or "failure"
	UserID  *int64
	IP      string
	Detail  map[string]string
}

// Recorder persists audit events.
type Recorder interface {
	Record(ctx context.Context, e Event)
}

// Store writes events to the audit_log table.
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

// Record stores e. It returns nothing on purpose: see the package comment.
func (s *Store) Record(ctx context.Context, e Event) {
	if !actionPattern.MatchString(e.Action) || (e.Outcome != "success" && e.Outcome != "failure") {
		slog.Warn("audit: malformed event dropped")
		return
	}

	detail, err := json.Marshal(sanitizeDetail(e.Detail))
	if err != nil {
		slog.Error("audit: detail not encodable", "action", e.Action)
		return
	}

	ip := ""
	if addr, err := netip.ParseAddr(e.IP); err == nil {
		ip = addr.String()
	}

	var actor pgtype.Int8
	if e.UserID != nil {
		actor = pgtype.Int8{Int64: *e.UserID, Valid: true}
	}

	// The request may already be cancelled (client hung up); the action still happened.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
	defer cancel()

	err = s.q.InsertAuditLog(writeCtx, dbsql.InsertAuditLogParams{
		OccurredAt:  pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
		ActorUserID: actor,
		Action:      e.Action,
		Outcome:     e.Outcome,
		Ip:          ip,
		Detail:      detail,
	})
	if err != nil {
		slog.Error("audit: write failed", "action", e.Action, "error", err)
	}
}

func sanitizeDetail(in map[string]string) map[string]string {
	keys := make([]string, 0, len(in))
	for k := range in {
		if k == "" || len(k) > maxDetailKeyLen || sensitiveKey(k) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic choice when over the cap
	if len(keys) > maxDetailKeys {
		keys = keys[:maxDetailKeys]
	}

	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = truncate(in[k], maxDetailValLen)
	}
	return out
}

func sensitiveKey(k string) bool {
	lower := strings.ToLower(k)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

// truncate cuts s to at most n bytes without splitting a UTF-8 character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
