package attachments

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pikipici/informatika-monitoring/backend/internal/dbsql"
)

// MaxPerMaterial mirrors the database trigger in migration 00007.
const MaxPerMaterial = 5

var (
	ErrMaterialNotFound = errors.New("attachments: material not found")
	ErrLimit            = errors.New("attachments: material already has the maximum number of attachments")
	ErrBusy             = errors.New("attachments: another upload is in progress")
)

// DB is what the service needs from a pool: queries and transactions.
type DB interface {
	dbsql.DBTX
	Begin(context.Context) (pgx.Tx, error)
}

// Attachment is a stored file and its metadata.
type Attachment struct {
	ID         int64
	MaterialID int64
	Key        string
	Name       string
	Kind       Kind
	Size       int64
	SHA256     []byte
	CreatedAt  time.Time
}

// Service ties stored files to material_attachments rows.
//
// Order of writes keeps the disk and the database consistent: a file is
// written before its row and removed if the row cannot be inserted; a row is
// deleted before its file. A crash between the two steps can leave an orphan
// file on disk, never a row without a file.
type Service struct {
	db    DB
	q     *dbsql.Queries
	files *Store
	now   func() time.Time
	slot  chan struct{}
}

// NewService builds a service that allows one upload at a time.
func NewService(db DB, files *Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{db: db, q: dbsql.New(db), files: files, now: now, slot: make(chan struct{}, 1)}
}

// MaxFile is the per-file size limit of the underlying store.
func (s *Service) MaxFile() int64 { return s.files.MaxFile() }

// Upload validates and stores r as a new attachment of materialID.
func (s *Service) Upload(ctx context.Context, materialID int64, r io.Reader, name string) (Attachment, error) {
	select {
	case s.slot <- struct{}{}:
		defer func() { <-s.slot }()
	default:
		return Attachment{}, ErrBusy
	}

	ok, err := s.q.MaterialExists(ctx, materialID)
	if err != nil {
		return Attachment{}, err
	}
	if !ok {
		return Attachment{}, ErrMaterialNotFound
	}
	usage, err := s.q.AttachmentUsage(ctx, materialID)
	if err != nil {
		return Attachment{}, err
	}
	if usage.MaterialCount >= MaxPerMaterial {
		return Attachment{}, ErrLimit
	}

	saved, err := s.files.Save(r, name, usage.TotalBytes)
	if err != nil {
		return Attachment{}, err
	}
	key, err := uuidFromKey(saved.Key)
	if err != nil {
		s.discard(saved.Key)
		return Attachment{}, err
	}
	row, err := s.q.InsertAttachment(context.WithoutCancel(ctx), dbsql.InsertAttachmentParams{
		MaterialID:   materialID,
		StorageKey:   key,
		OriginalName: saved.Name,
		Kind:         saved.Kind.String(),
		ContentType:  saved.Kind.ContentType(),
		SizeBytes:    saved.Size,
		Sha256:       saved.SHA256,
		CreatedAt:    pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
	})
	if err != nil {
		s.discard(saved.Key)
		return Attachment{}, insertError(err)
	}
	return fromRow(row), nil
}

// List returns the attachments of a material, oldest first.
func (s *Service) List(ctx context.Context, materialID int64) ([]Attachment, error) {
	ok, err := s.q.MaterialExists(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMaterialNotFound
	}
	rows, err := s.q.ListMaterialAttachments(ctx, materialID)
	if err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

// PublicList returns the attachments of a published material. A draft or
// missing material gives ErrMaterialNotFound, so drafts never leak.
func (s *Service) PublicList(ctx context.Context, materialID int64) ([]Attachment, error) {
	ok, err := s.q.PublishedMaterialExists(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMaterialNotFound
	}
	rows, err := s.q.ListMaterialAttachments(ctx, materialID)
	if err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

// PublicOpen returns an attachment of a published material and its open
// file. The caller closes the file. A row whose file is gone is reported as
// ErrNotFound and logged.
func (s *Service) PublicOpen(ctx context.Context, id int64) (Attachment, *os.File, error) {
	row, err := s.q.GetPublicAttachment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, nil, ErrNotFound
	}
	if err != nil {
		return Attachment{}, nil, err
	}
	a := fromRow(row)
	f, err := s.files.Open(a.Key)
	if errors.Is(err, ErrNotFound) {
		slog.Error("attachment file missing", "id", a.ID, "key", a.Key)
		return Attachment{}, nil, ErrNotFound
	}
	if err != nil {
		return Attachment{}, nil, err
	}
	return a, f, nil
}

// Delete removes one attachment: the row first, then its file.
func (s *Service) Delete(ctx context.Context, id int64) (Attachment, error) {
	row, err := s.q.DeleteAttachment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, err
	}
	a := fromRow(row)
	s.discard(a.Key)
	return a, nil
}

// DeleteMaterial deletes a material and the files of its attachments. The
// material row is locked first; the attachment trigger takes the same lock,
// so an upload racing with this delete either finishes before the keys are
// read or fails its insert and removes its own file.
func (s *Service) DeleteMaterial(ctx context.Context, id int64) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	q := s.q.WithTx(tx)
	title, err := q.LockMaterialTitle(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMaterialNotFound
	}
	if err != nil {
		return "", err
	}
	keys, err := q.MaterialAttachmentKeys(ctx, id)
	if err != nil {
		return "", err
	}
	if _, err := q.DeleteMaterial(ctx, id); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	for _, k := range keys {
		s.discard(keyFromUUID(k))
	}
	return title, nil
}

// discard removes a file whose row is gone (or never existed). Failure is
// logged with the key only; the file is then an orphan, not a broken row.
func (s *Service) discard(key string) {
	if err := s.files.Remove(key); err != nil {
		slog.Error("attachment file not removed", "key", key, "error", err)
	}
}

func insertError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch {
		case pg.Code == "23514" && pg.ConstraintName == "":
			// Raised by material_attachments_limit(), not a table constraint.
			return ErrLimit
		case pg.Code == "23503":
			return ErrMaterialNotFound
		}
	}
	return fmt.Errorf("attachments: insert: %w", err)
}

func fromRow(r dbsql.MaterialAttachment) Attachment {
	return Attachment{
		ID:         r.ID,
		MaterialID: r.MaterialID,
		Key:        keyFromUUID(r.StorageKey),
		Name:       r.OriginalName,
		Kind:       kindFromString(r.Kind),
		Size:       r.SizeBytes,
		SHA256:     r.Sha256,
		CreatedAt:  r.CreatedAt.Time,
	}
}

func kindFromString(s string) Kind {
	for _, k := range []Kind{PDF, PPT, PPTX} {
		if k.String() == s {
			return k
		}
	}
	return 0
}

func uuidFromKey(key string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if !ValidKey(key) {
		return u, fmt.Errorf("attachments: invalid key")
	}
	b, err := hex.DecodeString(key[0:8] + key[9:13] + key[14:18] + key[19:23] + key[24:36])
	if err != nil || len(b) != 16 {
		return u, fmt.Errorf("attachments: invalid key")
	}
	copy(u.Bytes[:], b)
	u.Valid = true
	return u, nil
}

func keyFromUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
