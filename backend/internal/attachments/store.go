package attachments

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits from PLAN.md section 0.
const (
	DefaultMaxFile  int64 = 25 << 20
	DefaultMaxTotal int64 = 2 << 30
	DefaultMinFree  int64 = 1 << 30

	maxNameRunes = 200
	fileMode     = 0o640
)

var (
	ErrTooLarge = errors.New("attachments: file too large")
	ErrQuota    = errors.New("attachments: storage quota exceeded")
	ErrDiskFull = errors.New("attachments: not enough free disk space")
	ErrName     = errors.New("attachments: unusable file name")
	ErrNotFound = errors.New("attachments: not found")
)

var keyPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidKey reports whether k is a storage key this package could have made.
func ValidKey(k string) bool { return keyPattern.MatchString(k) }

// Options configures a Store. Zero limits use the defaults.
type Options struct {
	Dir      string
	MaxFile  int64
	MaxTotal int64
	MinFree  int64
	// FreeBytes reports free space on the filesystem holding dir. Tests replace it.
	FreeBytes func(dir string) (uint64, error)
}

// Store keeps attachment files in one private directory, named by random key.
type Store struct {
	dir  string
	opts Options
}

// Saved describes a stored file.
type Saved struct {
	Key    string
	Kind   Kind
	Size   int64
	SHA256 []byte
	Name   string
}

// NewStore checks that dir exists, is a real directory, and is not writable or
// readable by others.
func NewStore(o Options) (*Store, error) {
	if strings.TrimSpace(o.Dir) == "" {
		return nil, errors.New("attachments: directory not configured")
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return nil, fmt.Errorf("attachments: directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("attachments: directory: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("attachments: directory path is not a directory")
	}
	if info.Mode().Perm()&0o007 != 0 {
		return nil, errors.New("attachments: directory must not be accessible to other users")
	}
	if o.MaxFile <= 0 {
		o.MaxFile = DefaultMaxFile
	}
	if o.MaxTotal <= 0 {
		o.MaxTotal = DefaultMaxTotal
	}
	if o.MinFree <= 0 {
		o.MinFree = DefaultMinFree
	}
	if o.FreeBytes == nil {
		o.FreeBytes = freeBytes
	}
	return &Store{dir: dir, opts: o}, nil
}

// MaxFile is the per-file size limit.
func (s *Store) MaxFile() int64 { return s.opts.MaxFile }

// Save streams r to disk, validates it, and commits it under a new key.
// currentTotal is the size of all files already stored (from the database).
// On any error nothing is left on disk.
func (s *Store) Save(r io.Reader, name string, currentTotal int64) (Saved, error) {
	clean, err := CleanName(name)
	if err != nil {
		return Saved{}, err
	}
	if kindFromName(clean) == 0 {
		return Saved{}, ErrType
	}
	if currentTotal >= s.opts.MaxTotal {
		return Saved{}, ErrQuota
	}
	if err := s.checkFree(s.opts.MaxFile); err != nil {
		return Saved{}, err
	}

	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return Saved{}, fmt.Errorf("attachments: temp file: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err := tmp.Chmod(fileMode); err != nil {
		return Saved{}, fmt.Errorf("attachments: chmod: %w", err)
	}

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, s.opts.MaxFile+1))
	if err != nil {
		return Saved{}, fmt.Errorf("attachments: receive: %w", err)
	}
	if n > s.opts.MaxFile {
		return Saved{}, ErrTooLarge
	}
	if currentTotal+n > s.opts.MaxTotal {
		return Saved{}, ErrQuota
	}

	kind, err := Detect(tmp, n, clean)
	if err != nil {
		return Saved{}, err
	}
	if err := tmp.Sync(); err != nil {
		return Saved{}, fmt.Errorf("attachments: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Saved{}, fmt.Errorf("attachments: close: %w", err)
	}

	key, err := newKey()
	if err != nil {
		return Saved{}, err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, key)); err != nil {
		return Saved{}, fmt.Errorf("attachments: commit: %w", err)
	}
	committed = true
	syncDir(s.dir)
	return Saved{Key: key, Kind: kind, Size: n, SHA256: h.Sum(nil), Name: clean}, nil
}

// Open returns the stored file for key.
func (s *Store) Open(key string) (*os.File, error) {
	if !ValidKey(key) {
		return nil, ErrNotFound
	}
	f, err := os.Open(filepath.Join(s.dir, key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// Remove deletes the file for key. A missing file is not an error.
func (s *Store) Remove(key string) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	err := os.Remove(filepath.Join(s.dir, key))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("attachments: remove: %w", err)
	}
	return nil
}

func (s *Store) checkFree(incoming int64) error {
	free, err := s.opts.FreeBytes(s.dir)
	if err != nil {
		return fmt.Errorf("attachments: free space: %w", err)
	}
	if free < uint64(s.opts.MinFree+incoming) {
		return ErrDiskFull
	}
	return nil
}

// CleanName turns a browser-supplied file name into safe display text: the
// last path element only, no control characters, trimmed, and at most 200
// characters with the extension kept.
func CleanName(name string) (string, error) {
	name = strings.ReplaceAll(name, `\`, "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if !utf8.ValidString(name) {
		return "", ErrName
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", ErrName
	}
	if runes := []rune(name); len(runes) > maxNameRunes {
		ext := []rune(filepath.Ext(name))
		if len(ext) >= maxNameRunes {
			return "", ErrName
		}
		name = string(runes[:maxNameRunes-len(ext)]) + string(ext)
	}
	return name, nil
}

func newKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("attachments: random key: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
