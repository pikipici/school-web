package attachments_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/attachments"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

type svcFixture struct {
	pool *pgxpool.Pool
	dir  string
	svc  *attachments.Service
}

func newSvcFixture(t *testing.T, maxTotal int64) *svcFixture {
	t.Helper()
	pool := testdb.OpenMigrated(t)
	dir := filepath.Join(t.TempDir(), "attachments")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	store, err := attachments.NewStore(attachments.Options{
		Dir:       dir,
		MaxTotal:  maxTotal,
		FreeBytes: func(string) (uint64, error) { return 100 << 30, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time { return time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC) }
	return &svcFixture{pool: pool, dir: dir, svc: attachments.NewService(pool, store, now)}
}

func (f *svcFixture) material(t *testing.T, title string) int64 {
	t.Helper()
	var id int64
	err := f.pool.QueryRow(context.Background(),
		`insert into materials (title, description, status) values ($1, '', 'draft') returning id`, title).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *svcFixture) files(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func (f *svcFixture) rows(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from material_attachments`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestServiceUploadStoresFileAndRow(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Algoritma")
	body := pdf()

	a, err := f.svc.Upload(ctx, mid, bytes.NewReader(body), `C:\Users\guru\Materi K1.pdf`)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	sum := sha256.Sum256(body)
	if a.ID < 1 || a.MaterialID != mid || a.Name != "Materi K1.pdf" || a.Kind != attachments.PDF ||
		a.Size != int64(len(body)) || !bytes.Equal(a.SHA256, sum[:]) || !attachments.ValidKey(a.Key) || a.CreatedAt.IsZero() {
		t.Fatalf("unexpected attachment: %+v", a)
	}
	if got := f.files(t); len(got) != 1 || got[0] != a.Key {
		t.Fatalf("files on disk = %v, want [%s]", got, a.Key)
	}
	stored, err := os.ReadFile(filepath.Join(f.dir, a.Key))
	if err != nil || !bytes.Equal(stored, body) {
		t.Fatalf("stored file differs (err=%v)", err)
	}

	list, err := f.svc.List(ctx, mid)
	if err != nil || len(list) != 1 || list[0].ID != a.ID || list[0].Key != a.Key {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if _, err := f.svc.List(ctx, mid+1000); !errors.Is(err, attachments.ErrMaterialNotFound) {
		t.Fatalf("List(missing) err = %v", err)
	}
}

func TestServiceUploadRejectionsLeaveNothingBehind(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Jaringan")

	cases := map[string]struct {
		material int64
		body     []byte
		name     string
		want     error
	}{
		"missing material": {mid + 1000, pdf(), "a.pdf", attachments.ErrMaterialNotFound},
		"fake pdf":         {mid, []byte("not a pdf at all"), "a.pdf", attachments.ErrType},
		"wrong extension":  {mid, pdf(), "a.pptx", attachments.ErrType},
		"bad name":         {mid, pdf(), "..", attachments.ErrName},
	}
	for name, tc := range cases {
		if _, err := f.svc.Upload(ctx, tc.material, bytes.NewReader(tc.body), tc.name); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
	if got := f.files(t); len(got) != 0 {
		t.Errorf("files left on disk: %v", got)
	}
	if n := f.rows(t); n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}

func TestServiceUploadLimitPerMaterial(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Basis Data")
	for i := 0; i < 5; i++ {
		if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf"); err != nil {
			t.Fatalf("upload %d: %v", i+1, err)
		}
	}
	if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf"); !errors.Is(err, attachments.ErrLimit) {
		t.Fatalf("sixth upload err = %v, want ErrLimit", err)
	}
	if got := f.files(t); len(got) != 5 {
		t.Fatalf("files = %d, want 5", len(got))
	}
	// Another material is unaffected.
	other := f.material(t, "Lain")
	if _, err := f.svc.Upload(ctx, other, bytes.NewReader(pdf()), "a.pdf"); err != nil {
		t.Fatalf("upload to other material: %v", err)
	}
}

func TestServiceQuotaCountsStoredBytes(t *testing.T) {
	size := int64(len(pdf()))
	f := newSvcFixture(t, 2*size+size/2)
	ctx := context.Background()
	mid := f.material(t, "Kuota")
	for i := 0; i < 2; i++ {
		if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf"); err != nil {
			t.Fatalf("upload %d: %v", i+1, err)
		}
	}
	if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf"); !errors.Is(err, attachments.ErrQuota) {
		t.Fatalf("third upload err = %v, want ErrQuota", err)
	}
	if got := f.files(t); len(got) != 2 {
		t.Fatalf("files = %d, want 2", len(got))
	}
}

func TestServiceInsertFailureRemovesFile(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Gagal")
	for _, stmt := range []string{
		`create function fail_insert() returns trigger language plpgsql as $$ begin raise exception 'boom'; end; $$`,
		`create trigger fail_insert before insert on material_attachments for each row execute function fail_insert()`,
	} {
		if _, err := f.pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf")
	if err == nil {
		t.Fatal("Upload succeeded despite failing insert")
	}
	for _, known := range []error{attachments.ErrLimit, attachments.ErrMaterialNotFound, attachments.ErrType} {
		if errors.Is(err, known) {
			t.Fatalf("insert failure mapped to %v", known)
		}
	}
	if got := f.files(t); len(got) != 0 {
		t.Fatalf("orphan file left on disk: %v", got)
	}
}

func TestServiceDeleteRemovesRowAndFile(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Hapus")
	a, err := f.svc.Upload(ctx, mid, bytes.NewReader(pptx(t)), "slide.pptx")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Delete(ctx, a.ID)
	if err != nil || got.ID != a.ID || got.MaterialID != mid || got.Name != "slide.pptx" {
		t.Fatalf("Delete = %+v, %v", got, err)
	}
	if files := f.files(t); len(files) != 0 {
		t.Fatalf("file still on disk: %v", files)
	}
	if n := f.rows(t); n != 0 {
		t.Fatalf("rows = %d", n)
	}
	if _, err := f.svc.Delete(ctx, a.ID); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatalf("second Delete err = %v, want ErrNotFound", err)
	}
}

func TestServiceDeleteMaterialRemovesItsFiles(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Materi Berlampiran")
	keep := f.material(t, "Tetap")
	for i := 0; i < 2; i++ {
		if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "a.pdf"); err != nil {
			t.Fatal(err)
		}
	}
	kept, err := f.svc.Upload(ctx, keep, bytes.NewReader(pdf()), "b.pdf")
	if err != nil {
		t.Fatal(err)
	}

	title, err := f.svc.DeleteMaterial(ctx, mid)
	if err != nil || title != "Materi Berlampiran" {
		t.Fatalf("DeleteMaterial = %q, %v", title, err)
	}
	if files := f.files(t); len(files) != 1 || files[0] != kept.Key {
		t.Fatalf("files after delete = %v, want only %s", files, kept.Key)
	}
	var n int
	if err := f.pool.QueryRow(ctx, `select count(*) from materials where id=$1`, mid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("material still present (n=%d, err=%v)", n, err)
	}
	if _, err := f.svc.DeleteMaterial(ctx, mid); !errors.Is(err, attachments.ErrMaterialNotFound) {
		t.Fatalf("second DeleteMaterial err = %v", err)
	}
}

// signalReader blocks on its first Read until released, telling the test the
// upload slot is held.
type signalReader struct {
	started chan struct{}
	release chan struct{}
	r       io.Reader
	once    bool
}

func (s *signalReader) Read(p []byte) (int, error) {
	if !s.once {
		s.once = true
		close(s.started)
		<-s.release
	}
	return s.r.Read(p)
}

func TestServiceAllowsOneUploadAtATime(t *testing.T) {
	f := newSvcFixture(t, 0)
	ctx := context.Background()
	mid := f.material(t, "Serentak")
	slow := &signalReader{started: make(chan struct{}), release: make(chan struct{}), r: bytes.NewReader(pdf())}
	done := make(chan error, 1)
	go func() {
		_, err := f.svc.Upload(ctx, mid, slow, "a.pdf")
		done <- err
	}()
	select {
	case <-slow.started:
	case <-time.After(10 * time.Second):
		t.Fatal("first upload never started reading")
	}
	if _, err := f.svc.Upload(ctx, mid, strings.NewReader(string(pdf())), "b.pdf"); !errors.Is(err, attachments.ErrBusy) {
		t.Errorf("concurrent upload err = %v, want ErrBusy", err)
	}
	close(slow.release)
	if err := <-done; err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if _, err := f.svc.Upload(ctx, mid, bytes.NewReader(pdf()), "c.pdf"); err != nil {
		t.Fatalf("upload after slot freed: %v", err)
	}
}
