package attachments_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/pikipici/informatika-monitoring/backend/internal/attachments"
)

// ---- fixtures ---------------------------------------------------------------

func pdf() []byte {
	return []byte("%PDF-1.7\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [] /Count 0 >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
}

const pptxTypes = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/></Types>`
const pptmTypes = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.ms-powerpoint.presentation.macroEnabled.main+xml"/></Types>`
const docxTypes = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pptx(t *testing.T) []byte {
	return zipOf(t, map[string]string{
		"[Content_Types].xml":   pptxTypes,
		"_rels/.rels":           "<Relationships/>",
		"ppt/presentation.xml":  "<p:presentation/>",
		"ppt/slides/slide1.xml": "<p:sld/>",
	})
}

// cfb builds a minimal but structurally valid Compound File (OLE2) with a root
// entry and the given stream names, like a legacy .ppt.
func cfb(t *testing.T, names ...string) []byte {
	t.Helper()
	const sector = 512
	const endOfChain, fatSect, freeSect = 0xFFFFFFFE, 0xFFFFFFFD, 0xFFFFFFFF
	entries := append([]string{"Root Entry"}, names...)
	dirSectors := (len(entries)*128 + sector - 1) / sector

	header := make([]byte, sector)
	copy(header, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	binary.LittleEndian.PutUint16(header[0x18:], 0x003E)
	binary.LittleEndian.PutUint16(header[0x1A:], 0x0003)
	binary.LittleEndian.PutUint16(header[0x1C:], 0xFFFE)
	binary.LittleEndian.PutUint16(header[0x1E:], 9) // 512-byte sectors
	binary.LittleEndian.PutUint16(header[0x20:], 6)
	binary.LittleEndian.PutUint32(header[0x2C:], 1) // one FAT sector
	binary.LittleEndian.PutUint32(header[0x30:], 1) // directory starts at sector 1
	binary.LittleEndian.PutUint32(header[0x38:], 4096)
	binary.LittleEndian.PutUint32(header[0x3C:], endOfChain)
	binary.LittleEndian.PutUint32(header[0x44:], endOfChain)
	for i := 0; i < 109; i++ {
		binary.LittleEndian.PutUint32(header[0x4C+i*4:], freeSect)
	}
	binary.LittleEndian.PutUint32(header[0x4C:], 0) // FAT is sector 0

	fat := make([]byte, sector)
	for i := 0; i < sector/4; i++ {
		binary.LittleEndian.PutUint32(fat[i*4:], freeSect)
	}
	binary.LittleEndian.PutUint32(fat[0:], fatSect)
	for i := 0; i < dirSectors; i++ {
		next := uint32(i + 2)
		if i == dirSectors-1 {
			next = endOfChain
		}
		binary.LittleEndian.PutUint32(fat[(1+i)*4:], next)
	}

	dir := make([]byte, dirSectors*sector)
	for i, name := range entries {
		e := dir[i*128 : (i+1)*128]
		u := utf16.Encode([]rune(name))
		for j, c := range u {
			binary.LittleEndian.PutUint16(e[j*2:], c)
		}
		binary.LittleEndian.PutUint16(e[0x40:], uint16((len(u)+1)*2))
		typ := byte(2)
		if i == 0 {
			typ = 5
		}
		e[0x42] = typ
		binary.LittleEndian.PutUint32(e[0x44:], freeSect)
		binary.LittleEndian.PutUint32(e[0x48:], freeSect)
		binary.LittleEndian.PutUint32(e[0x4C:], freeSect)
		if i == 0 && len(entries) > 1 {
			binary.LittleEndian.PutUint32(e[0x4C:], 1) // root's child
		}
		if i > 0 && i < len(entries)-1 {
			binary.LittleEndian.PutUint32(e[0x48:], uint32(i+1)) // right sibling chain
		}
		binary.LittleEndian.PutUint32(e[0x74:], endOfChain)
	}
	return append(append(header, fat...), dir...)
}

func ppt(t *testing.T) []byte {
	return cfb(t, "PowerPoint Document", "Current User", "\x05SummaryInformation")
}

// ---- Detect -----------------------------------------------------------------

func detect(t *testing.T, name string, data []byte) (attachments.Kind, error) {
	t.Helper()
	return attachments.Detect(bytes.NewReader(data), int64(len(data)), name)
}

func TestDetectAcceptsGenuineFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		file string
		data []byte
		want attachments.Kind
	}{
		"pdf":               {"Modul.pdf", pdf(), attachments.PDF},
		"pptx":              {"Slide.pptx", pptx(t), attachments.PPTX},
		"ppt":               {"Lama.ppt", ppt(t), attachments.PPT},
		"upper-case suffix": {"MODUL.PDF", pdf(), attachments.PDF},
		"pdf 2.0":           {"baru.pdf", append([]byte("%PDF-2.0\n"), pdf()[9:]...), attachments.PDF},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := detect(t, tc.file, tc.data)
			if err != nil || got != tc.want {
				t.Fatalf("Detect = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestKindMetadata(t *testing.T) {
	for k, want := range map[attachments.Kind][2]string{
		attachments.PDF:  {"pdf", "application/pdf"},
		attachments.PPT:  {"ppt", "application/vnd.ms-powerpoint"},
		attachments.PPTX: {"pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
	} {
		if k.String() != want[0] || k.ContentType() != want[1] {
			t.Errorf("%v: %q %q", k, k.String(), k.ContentType())
		}
	}
}

func TestDetectRejectsWrongOrDangerousFiles(t *testing.T) {
	scriptInPDFName := []byte("<html><script>alert(1)</script></html>")
	cases := map[string]struct {
		file string
		data []byte
		want error
	}{
		"html renamed pdf":              {"a.pdf", scriptInPDFName, attachments.ErrType},
		"pdf named pptx":                {"a.pptx", pdf(), attachments.ErrType},
		"pptx named pdf":                {"a.pdf", pptx(t), attachments.ErrType},
		"ppt named pptx":                {"a.pptx", ppt(t), attachments.ErrType},
		"pdf marker not at start":       {"a.pdf", append([]byte("junk"), pdf()...), attachments.ErrType},
		"empty":                         {"a.pdf", nil, attachments.ErrType},
		"docx renamed pptx":             {"a.pptx", zipOf(t, map[string]string{"[Content_Types].xml": docxTypes, "word/document.xml": "<w/>"}), attachments.ErrType},
		"zip without content types":     {"a.pptx", zipOf(t, map[string]string{"ppt/presentation.xml": "<p/>"}), attachments.ErrType},
		"zip without presentation part": {"a.pptx", zipOf(t, map[string]string{"[Content_Types].xml": pptxTypes}), attachments.ErrType},
		"macro-enabled content type":    {"a.pptx", zipOf(t, map[string]string{"[Content_Types].xml": pptmTypes, "ppt/presentation.xml": "<p/>"}), attachments.ErrMacro},
		"vba project in pptx":           {"a.pptx", zipOf(t, map[string]string{"[Content_Types].xml": pptxTypes, "ppt/presentation.xml": "<p/>", "ppt/vbaProject.bin": "x"}), attachments.ErrMacro},
		"truncated zip":                 {"a.pptx", pptx(t)[:40], attachments.ErrType},
		"word document in OLE":          {"a.ppt", cfb(t, "WordDocument", "1Table"), attachments.ErrType},
		"ppt with VBA storage":          {"a.ppt", cfb(t, "PowerPoint Document", "_VBA_PROJECT_CUR"), attachments.ErrMacro},
		"ole header only":               {"a.ppt", ppt(t)[:512], attachments.ErrType},
		"pptm extension":                {"a.pptm", pptx(t), attachments.ErrType},
		"ppsx extension":                {"a.ppsx", pptx(t), attachments.ErrType},
		"no extension":                  {"modul", pdf(), attachments.ErrType},
		"double extension ending exe":   {"modul.pdf.exe", pdf(), attachments.ErrType},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := detect(t, tc.file, tc.data); !errors.Is(err, tc.want) {
				t.Fatalf("Detect = %v; want %v", err, tc.want)
			}
		})
	}
}

func TestDetectSurvivesHostileCFB(t *testing.T) {
	good := ppt(t)
	// Point the directory chain at itself (a loop) and at an absurd sector.
	loop := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(loop[512+4:], 1)
	far := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(far[0x30:], 0x7FFFFFF0)
	hugeShift := append([]byte(nil), good...)
	binary.LittleEndian.PutUint16(hugeShift[0x1E:], 30)
	for name, data := range map[string][]byte{"loop": loop, "far sector": far, "huge sector shift": hugeShift} {
		t.Run(name, func(t *testing.T) {
			if _, err := detect(t, "a.ppt", data); !errors.Is(err, attachments.ErrType) {
				t.Fatalf("Detect = %v; want ErrType", err)
			}
		})
	}
}

func TestDetectDoesNotInflateZipBombs(t *testing.T) {
	// A 50 MB run of zeros compresses to a few KB; only the small content-types
	// part may ever be read.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range map[string]string{"[Content_Types].xml": pptxTypes, "ppt/presentation.xml": "<p/>"} {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(body))
	}
	f, _ := w.Create("ppt/media/big.bin")
	_, _ = f.Write(make([]byte, 50<<20))
	_ = w.Close()
	if buf.Len() > 1<<20 {
		t.Fatalf("fixture not compressed: %d bytes", buf.Len())
	}
	if k, err := detect(t, "a.pptx", buf.Bytes()); err != nil || k != attachments.PPTX {
		t.Fatalf("Detect = %v, %v", k, err)
	}

	hugeTypes := zipOf(t, map[string]string{"[Content_Types].xml": pptxTypes + strings.Repeat(" ", 2<<20), "ppt/presentation.xml": "<p/>"})
	if _, err := detect(t, "a.pptx", hugeTypes); !errors.Is(err, attachments.ErrType) {
		t.Fatalf("oversized content-types part = %v; want ErrType", err)
	}
}

// ---- CleanName --------------------------------------------------------------

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"Modul Graf.pdf":                   "Modul Graf.pdf",
		`C:\Users\guru\Desktop\Slide.pptx`: "Slide.pptx",
		"/etc/../secret/Slide.pptx":        "Slide.pptx",
		"  spasi  .pdf ":                   "spasi  .pdf",
		"tab\there\n.pdf":                  "tabhere.pdf",
		"日本語の資料.pdf":                       "日本語の資料.pdf",
		strings.Repeat("a", 300) + ".pptx": strings.Repeat("a", 195) + ".pptx",
		strings.Repeat("é", 250) + ".pdf":  strings.Repeat("é", 196) + ".pdf",
	} {
		got, err := attachments.CleanName(in)
		if err != nil || got != want {
			t.Errorf("CleanName(%q) = %q, %v; want %q", in, got, err, want)
		}
		if n := len([]rune(got)); n > 200 {
			t.Errorf("CleanName(%q) is %d characters", in, n)
		}
	}
	for _, bad := range []string{"", "   ", "/", `\`, "..", "\x00\x01", "folder/"} {
		if _, err := attachments.CleanName(bad); !errors.Is(err, attachments.ErrName) {
			t.Errorf("CleanName(%q) err = %v; want ErrName", bad, err)
		}
	}
}

// ---- Store -------------------------------------------------------------------

type fixture struct {
	dir   string
	store *attachments.Store
	free  uint64
}

func newStore(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), free: 50 << 30}
	if err := os.Chmod(f.dir, 0o750); err != nil { // same as production
		t.Fatal(err)
	}
	s, err := attachments.NewStore(attachments.Options{
		Dir:       f.dir,
		MaxFile:   1 << 20, // 1 MiB keeps tests fast
		MaxTotal:  3 << 20,
		MinFree:   10 << 20,
		FreeBytes: func(string) (uint64, error) { return f.free, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.store = s
	return f
}

func (f *fixture) files(t *testing.T) []string {
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

func TestSaveStoresUnderRandomKeyWithHashAndPrivateMode(t *testing.T) {
	f := newStore(t)
	data := pptx(t)

	saved, err := f.store.Save(bytes.NewReader(data), "Slide.pptx", 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.Kind != attachments.PPTX || saved.Size != int64(len(data)) {
		t.Fatalf("saved = %+v", saved)
	}
	if want := sha256.Sum256(data); !bytes.Equal(saved.SHA256, want[:]) {
		t.Fatal("hash does not match file content")
	}
	if !attachments.ValidKey(saved.Key) {
		t.Fatalf("key %q is not a canonical UUIDv4", saved.Key)
	}
	names := f.files(t)
	if len(names) != 1 || names[0] != saved.Key {
		t.Fatalf("directory = %v; want only %s (no extension, no temp leftovers)", names, saved.Key)
	}
	info, err := os.Stat(filepath.Join(f.dir, saved.Key))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o027 != 0 {
		t.Fatalf("file mode %v grants group-write or other access", perm)
	}

	other, err := f.store.Save(bytes.NewReader(data), "Slide.pptx", saved.Size)
	if err != nil || other.Key == saved.Key {
		t.Fatalf("second save key = %q, %v; keys must be unique", other.Key, err)
	}

	r, err := f.store.Open(saved.Key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if !bytes.Equal(got, data) {
		t.Fatal("Open returned different bytes")
	}
}

func TestSaveRejectsAndLeavesNothingBehind(t *testing.T) {
	cases := map[string]struct {
		data  []byte
		name  string
		total int64
		free  uint64
		want  error
	}{
		"too large":        {bytes.Repeat([]byte("%PDF-1.4 "), (1<<20)/9+10), "a.pdf", 0, 50 << 30, attachments.ErrTooLarge},
		"wrong type":       {[]byte("<html></html>"), "a.pdf", 0, 50 << 30, attachments.ErrType},
		"macro":            {nil, "a.pptx", 0, 50 << 30, attachments.ErrMacro},
		"quota exceeded":   {pdf(), "a.pdf", 3<<20 - 10, 50 << 30, attachments.ErrQuota},
		"disk nearly full": {pdf(), "a.pdf", 0, 10<<20 + 100, attachments.ErrDiskFull},
		"bad name":         {pdf(), "///", 0, 50 << 30, attachments.ErrName},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newStore(t)
			f.free = tc.free
			data := tc.data
			if name == "macro" {
				data = zipOf(t, map[string]string{"[Content_Types].xml": pptmTypes, "ppt/presentation.xml": "<p/>"})
			}
			_, err := f.store.Save(bytes.NewReader(data), tc.name, tc.total)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Save = %v; want %v", err, tc.want)
			}
			if left := f.files(t); len(left) != 0 {
				t.Fatalf("files left after a rejected upload: %v", left)
			}
		})
	}
}

type failingReader struct{ after int }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.after <= 0 {
		return 0, errors.New("connection reset")
	}
	n := copy(p, bytes.Repeat([]byte("x"), min(len(p), r.after)))
	r.after -= n
	return n, nil
}

func TestSaveCleansUpWhenTheUploadBreaks(t *testing.T) {
	f := newStore(t)
	if _, err := f.store.Save(&failingReader{after: 5000}, "a.pdf", 0); err == nil {
		t.Fatal("Save succeeded on a broken stream")
	}
	if left := f.files(t); len(left) != 0 {
		t.Fatalf("temp file left behind: %v", left)
	}
}

func TestOpenAndRemoveRefuseNonKeys(t *testing.T) {
	f := newStore(t)
	if err := os.WriteFile(filepath.Join(filepath.Dir(f.dir), "secret"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"", "../secret", "secret", "00000000-0000-4000-8000-00000000000G", "00000000-0000-4000-8000-000000000000/../../secret", "00000000-0000-4000-8000-000000000000 "} {
		if _, err := f.store.Open(k); !errors.Is(err, attachments.ErrNotFound) {
			t.Errorf("Open(%q) = %v; want ErrNotFound", k, err)
		}
		if err := f.store.Remove(k); !errors.Is(err, attachments.ErrNotFound) {
			t.Errorf("Remove(%q) = %v; want ErrNotFound", k, err)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f.dir), "secret")); err != nil {
		t.Fatal("a file outside the store was removed")
	}
}

func TestRemoveIsIdempotent(t *testing.T) {
	f := newStore(t)
	saved, err := f.store.Save(bytes.NewReader(pdf()), "a.pdf", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Remove(saved.Key); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Remove(saved.Key); err != nil {
		t.Fatalf("second Remove = %v; want nil", err)
	}
	if _, err := f.store.Open(saved.Key); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatalf("Open after Remove = %v; want ErrNotFound", err)
	}
}

func TestNewStoreRequiresAPrivateExistingDirectory(t *testing.T) {
	if _, err := attachments.NewStore(attachments.Options{Dir: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("NewStore accepted a missing directory")
	}
	if _, err := attachments.NewStore(attachments.Options{Dir: ""}); err == nil {
		t.Fatal("NewStore accepted an empty directory")
	}
	open := t.TempDir()
	if err := os.Chmod(open, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := attachments.NewStore(attachments.Options{Dir: open}); err == nil {
		t.Fatal("NewStore accepted a world-writable directory")
	}
}

func TestDefaultLimitsMatchThePlan(t *testing.T) {
	if attachments.DefaultMaxFile != 25<<20 || attachments.DefaultMaxTotal != 2<<30 || attachments.DefaultMinFree != 1<<30 {
		t.Fatalf("defaults = %d / %d / %d", attachments.DefaultMaxFile, attachments.DefaultMaxTotal, attachments.DefaultMinFree)
	}
}

// testdata/real-python-pptx.pptx was produced by python-pptx 1.0.2 from its
// built-in PowerPoint template, so it has the full real-world package layout.
func TestDetectAcceptsRealPPTXAndRejectsItRenamed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "real-python-pptx.pptx"))
	if err != nil {
		t.Fatal(err)
	}
	if k, err := detect(t, "Struktur Data.pptx", data); err != nil || k != attachments.PPTX {
		t.Fatalf("real pptx: %v, %v", k, err)
	}
	for _, name := range []string{"a.pdf", "a.ppt", "a.pptm", "a.docx"} {
		if _, err := detect(t, name, data); !errors.Is(err, attachments.ErrType) {
			t.Errorf("real pptx named %s: %v; want ErrType", name, err)
		}
	}
}
