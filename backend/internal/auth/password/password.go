// Package password hashes and verifies admin passwords with Argon2id and
// enforces the password policy. It never logs or returns plaintext passwords.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	// MaxLength caps input size (bytes) so oversized passwords cannot be used
	// to burn CPU or memory.
	MaxLength = 128
	// MinLength is the minimum number of characters (runes).
	MinLength = 12

	minDistinct = 5

	// Upper bounds applied to parameters read from a stored hash, so a
	// tampered row cannot make Verify allocate unbounded memory or time.
	maxMemoryKiB   = 256 * 1024
	maxIterations  = 10
	maxParallelism = 16
	minSaltLength  = 8
	maxSaltLength  = 64
	minKeyLength   = 16
	maxKeyLength   = 64
)

var (
	ErrTooShort         = errors.New("password: too short")
	ErrTooLong          = errors.New("password: too long")
	ErrCommon           = errors.New("password: too common")
	ErrContainsUsername = errors.New("password: contains username")
	ErrLowVariety       = errors.New("password: too few distinct characters")

	ErrInvalidHash = errors.New("password: invalid hash encoding")
)

// Params are the Argon2id cost settings. Memory is in KiB.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams returns production settings (64 MiB, 3 passes, 2 lanes),
// above the OWASP minimum of 19 MiB / 2 passes / 1 lane.
func DefaultParams() Params {
	return Params{Memory: 64 * 1024, Iterations: 3, Parallelism: 2, SaltLength: 16, KeyLength: 32}
}

// Hasher hashes and verifies passwords with fixed parameters.
type Hasher struct {
	p Params

	dummyOnce sync.Once
	dummy     string
}

// NewHasher validates params and returns a Hasher.
func NewHasher(p Params) (*Hasher, error) {
	switch {
	case p.Memory == 0 || p.Memory > maxMemoryKiB:
		return nil, errors.New("password: memory out of range")
	case p.Iterations == 0 || p.Iterations > maxIterations:
		return nil, errors.New("password: iterations out of range")
	case p.Parallelism == 0 || p.Parallelism > maxParallelism:
		return nil, errors.New("password: parallelism out of range")
	case p.SaltLength < minSaltLength || p.SaltLength > maxSaltLength:
		return nil, errors.New("password: salt length out of range")
	case p.KeyLength < minKeyLength || p.KeyLength > maxKeyLength:
		return nil, errors.New("password: key length out of range")
	}
	return &Hasher{p: p}, nil
}

// Hash returns a PHC-format Argon2id hash with a random salt.
func (h *Hasher) Hash(plain string) (string, error) {
	if len(plain) > MaxLength {
		return "", ErrTooLong
	}
	salt := make([]byte, h.p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: read random salt: %w", err)
	}
	return encode(h.p, salt, derive(plain, salt, h.p)), nil
}

// Verify reports whether plain matches encoded. needsRehash is true when the
// stored hash uses weaker parameters than the Hasher's current ones. A
// non-matching password returns (false, false, nil); a malformed or hostile
// encoding returns an error.
func (h *Hasher) Verify(plain, encoded string) (match, needsRehash bool, err error) {
	if len(plain) > MaxLength {
		return false, false, ErrTooLong
	}
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false, false, err
	}
	got := derive(plain, salt, p)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false, nil
	}
	return true, p != h.p, nil
}

// DummyVerify burns roughly the same time as a real Verify. Call it when the
// account does not exist so response time does not reveal valid usernames.
func (h *Hasher) DummyVerify(plain string) {
	h.dummyOnce.Do(func() {
		h.dummy, _ = h.Hash("dummy-password-for-timing-only")
	})
	_, _, _ = h.Verify(plain, h.dummy)
}

func derive(plain string, salt []byte, p Params) []byte {
	return argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
}

func encode(p Params, salt, key []byte) string {
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64.EncodeToString(salt), b64.EncodeToString(key))
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if n, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || n != 1 || version != argon2.Version {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var p Params
	var parallelism uint32
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &parallelism); err != nil || n != 3 {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if parallelism == 0 || parallelism > maxParallelism {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.Parallelism = uint8(parallelism)

	b64 := base64.RawStdEncoding.Strict()
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLength, p.KeyLength = uint32(len(salt)), uint32(len(key))

	if p.Memory == 0 || p.Memory > maxMemoryKiB ||
		p.Iterations == 0 || p.Iterations > maxIterations ||
		p.SaltLength < minSaltLength || p.SaltLength > maxSaltLength ||
		p.KeyLength < minKeyLength || p.KeyLength > maxKeyLength {
		return Params{}, nil, nil, ErrInvalidHash
	}
	return p, salt, key, nil
}

// commonPasswords holds lower-case passwords that meet the length rule but are
// widely guessed. Extend as needed; entries must be lower-case.
var commonPasswords = map[string]struct{}{
	"password1234":    {},
	"password12345":   {},
	"password123456":  {},
	"123456789012":    {},
	"1234567890123":   {},
	"qwertyuiop12":    {},
	"qwerty123456":    {},
	"qwertyuiopas":    {},
	"iloveyou1234":    {},
	"administrator":   {},
	"admin1234567":    {},
	"admin12345678":   {},
	"welcome12345":    {},
	"letmein12345":    {},
	"changeme1234":    {},
	"abcdefghijkl":    {},
	"abc123456789":    {},
	"informatika123":  {},
	"informatika1234": {},
	"sekolah12345":    {},
	"indonesia1234":   {},
	"indonesia2026":   {},
	"password2026":    {},
	"passw0rd1234":    {},
}

// ValidatePolicy checks a new password. Errors never contain the password.
func ValidatePolicy(plain, username string) error {
	if len(plain) > MaxLength {
		return ErrTooLong
	}
	if utf8.RuneCountInString(plain) < MinLength {
		return ErrTooShort
	}

	lower := strings.ToLower(plain)
	if u := strings.ToLower(strings.TrimSpace(username)); len(u) >= 3 && strings.Contains(lower, u) {
		return ErrContainsUsername
	}
	if _, bad := commonPasswords[lower]; bad {
		return ErrCommon
	}

	distinct := map[rune]struct{}{}
	for _, r := range lower {
		distinct[r] = struct{}{}
	}
	if len(distinct) < minDistinct {
		return ErrLowVariety
	}
	return nil
}
