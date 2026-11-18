package progress

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
)

// ErrConfirm means the typed group name did not match, so nothing was removed.
var ErrConfirm = errors.New("progress: confirmation name does not match")

// Purged reports what PurgeGroup removed.
type Purged struct {
	Name     string
	Rounds   int64
	Findings int64
	Notebook bool
}

// PurgeGroup permanently removes a group together with its submission rounds,
// notebook record, and findings (published ones included). The caller must
// type the group's exact name (surrounding spaces ignored). Everything,
// including the audit row, happens in one transaction: a wrong name, a
// missing group, or a failed audit write leaves every row in place.
func (s *Store) PurgeGroup(ctx context.Context, id int64, confirmName string, a Actor) (Purged, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Purged{}, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	q := s.q.WithTx(tx)
	g, err := q.LockGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Purged{}, ErrNotFound
	}
	if err != nil {
		return Purged{}, err
	}
	if strings.TrimSpace(confirmName) != g.Name {
		return Purged{}, ErrConfirm
	}
	p := Purged{Name: g.Name}
	if p.Findings, err = q.PurgeGroupFindings(ctx, id); err != nil {
		return Purged{}, err
	}
	if p.Rounds, err = q.PurgeGroupRounds(ctx, id); err != nil {
		return Purged{}, err
	}
	nb, err := q.PurgeGroupNotebook(ctx, id)
	if err != nil {
		return Purged{}, err
	}
	p.Notebook = nb > 0
	if _, err := q.DeleteGroup(ctx, id); err != nil {
		return Purged{}, deleteError(err)
	}
	uid := a.UserID
	detail := map[string]string{
		"entityId": strconv.FormatInt(id, 10),
		"title":    g.Name,
		"rounds":   strconv.FormatInt(p.Rounds, 10),
		"findings": strconv.FormatInt(p.Findings, 10),
		"notebook": strconv.FormatBool(p.Notebook),
	}
	if err := audit.Write(ctx, q, s.now(), audit.Event{Action: "progress.group_purged", Outcome: "success", UserID: &uid, IP: a.IP, Detail: detail}); err != nil {
		return Purged{}, fmt.Errorf("progress: audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Purged{}, err
	}
	return p, nil
}
