package password_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
)

// cheap keeps tests fast; production uses password.DefaultParams().
func cheap() password.Params {
	return password.Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

func newHasher(t *testing.T, p password.Params) *password.Hasher {
	t.Helper()
	h, err := password.NewHasher(p)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return h
}

func TestHashProducesArgon2idPHCString(t *testing.T) {
	h := newHasher(t, cheap())
	enc, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("unexpected encoding prefix: %q", enc)
	}
	if strings.Contains(enc, "correct horse") {
		t.Fatal("encoded hash contains the plaintext password")
	}
}

func TestHashUsesRandomSalt(t *testing.T) {
	h := newHasher(t, cheap())
	a, _ := h.Hash("same password value")
	b, _ := h.Hash("same password value")
	if a == b {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestVerify(t *testing.T) {
	h := newHasher(t, cheap())
	enc, _ := h.Hash("Sandi Rahasia 2026!")

	t.Run("correct password matches", func(t *testing.T) {
		ok, _, err := h.Verify("Sandi Rahasia 2026!", enc)
		if err != nil || !ok {
			t.Fatalf("Verify = %v, %v; want true, nil", ok, err)
		}
	})
	t.Run("wrong password does not match", func(t *testing.T) {
		ok, _, err := h.Verify("sandi rahasia 2026!", enc)
		if err != nil || ok {
			t.Fatalf("Verify = %v, %v; want false, nil", ok, err)
		}
	})
	t.Run("unicode password works", func(t *testing.T) {
		u, _ := h.Hash("kata sandi 日本語 ñandú")
		ok, _, err := h.Verify("kata sandi 日本語 ñandú", u)
		if err != nil || !ok {
			t.Fatalf("Verify unicode = %v, %v", ok, err)
		}
	})
}

func TestVerifyRejectsMalformedEncodings(t *testing.T) {
	h := newHasher(t, cheap())
	valid, _ := h.Hash("whatever password")

	cases := map[string]string{
		"empty":             "",
		"garbage":           "not-a-hash",
		"wrong algorithm":   strings.Replace(valid, "argon2id", "argon2i", 1),
		"wrong version":     strings.Replace(valid, "v=19", "v=16", 1),
		"missing parts":     "$argon2id$v=19$m=64,t=1,p=1$c2FsdA",
		"bad base64":        "$argon2id$v=19$m=64,t=1,p=1$!!!$!!!",
		"absurd memory":     "$argon2id$v=19$m=4294967295,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"absurd iterations": "$argon2id$v=19$m=64,t=4294967295,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"zero parallelism":  "$argon2id$v=19$m=64,t=1,p=0$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
	}
	for name, enc := range cases {
		t.Run(name, func(t *testing.T) {
			ok, _, err := h.Verify("whatever password", enc)
			if err == nil || ok {
				t.Fatalf("Verify(%q) = %v, %v; want false and an error", name, ok, err)
			}
		})
	}
}

func TestVerifyReportsRehashWhenParamsAreWeaker(t *testing.T) {
	weak := newHasher(t, cheap())
	enc, _ := weak.Hash("a decent password!!")

	stronger := cheap()
	stronger.Iterations = 2
	strong := newHasher(t, stronger)

	ok, rehash, err := strong.Verify("a decent password!!", enc)
	if err != nil || !ok {
		t.Fatalf("Verify = %v, %v", ok, err)
	}
	if !rehash {
		t.Fatal("expected needsRehash=true for hash made with weaker params")
	}

	ok, rehash, err = weak.Verify("a decent password!!", enc)
	if err != nil || !ok || rehash {
		t.Fatalf("same params: ok=%v rehash=%v err=%v; want true,false,nil", ok, rehash, err)
	}
}

func TestNewHasherRejectsUnsafeParams(t *testing.T) {
	bad := []password.Params{
		{Memory: 0, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32},
		{Memory: 64, Iterations: 0, Parallelism: 1, SaltLength: 16, KeyLength: 32},
		{Memory: 64, Iterations: 1, Parallelism: 0, SaltLength: 16, KeyLength: 32},
		{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 4, KeyLength: 32},
		{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 8},
	}
	for i, p := range bad {
		if _, err := password.NewHasher(p); err == nil {
			t.Errorf("case %d: NewHasher accepted unsafe params %+v", i, p)
		}
	}
}

func TestDefaultParamsMeetModernMinimum(t *testing.T) {
	p := password.DefaultParams()
	if p.Memory < 19*1024 || p.Iterations < 2 {
		t.Fatalf("default params too weak: %+v", p)
	}
	if _, err := password.NewHasher(p); err != nil {
		t.Fatalf("default params rejected: %v", err)
	}
}

func TestDummyVerifyNeverMatches(t *testing.T) {
	h := newHasher(t, cheap())
	h.DummyVerify("anything at all")
	h.DummyVerify("")
}

func TestHashRejectsOverlongPassword(t *testing.T) {
	h := newHasher(t, cheap())
	if _, err := h.Hash(strings.Repeat("a", password.MaxLength+1)); !errors.Is(err, password.ErrTooLong) {
		t.Fatalf("Hash overlong = %v; want ErrTooLong", err)
	}
	ok, _, err := h.Verify(strings.Repeat("a", password.MaxLength+1), "$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g")
	if ok || !errors.Is(err, password.ErrTooLong) {
		t.Fatalf("Verify overlong = %v, %v; want false, ErrTooLong", ok, err)
	}
}

func TestValidatePolicy(t *testing.T) {
	cases := []struct {
		name     string
		password string
		username string
		want     error
	}{
		{"strong passphrase accepted", "kopi-hangat-di-pagi-hari", "fikri", nil},
		{"mixed password accepted", "T4hun-Ajaran#2627x", "fikri", nil},
		{"exactly 12 chars accepted", "abcdefgh1234", "fikri", nil},
		{"too short", "Short1!", "fikri", password.ErrTooShort},
		{"11 chars too short", "abcdefgh123", "fikri", password.ErrTooShort},
		{"unicode counted by character", "日本語日本語日本語日本語", "fikri", password.ErrLowVariety},
		{"too long", strings.Repeat("ab", 70), "fikri", password.ErrTooLong},
		{"common password", "password1234", "fikri", password.ErrCommon},
		{"common password case-insensitive", "PassWord1234", "fikri", password.ErrCommon},
		{"contains username", "myfikri-secret-99", "fikri", password.ErrContainsUsername},
		{"contains username case-insensitive", "My-FIKRI-secret-99", "fikri", password.ErrContainsUsername},
		{"single repeated character", "aaaaaaaaaaaaaa", "fikri", password.ErrLowVariety},
		{"few distinct characters", "abababababab", "fikri", password.ErrLowVariety},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := password.ValidatePolicy(tc.password, tc.username)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("ValidatePolicy = %v; want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("ValidatePolicy = %v; want %v", err, tc.want)
			}
			if strings.Contains(err.Error(), tc.password) {
				t.Fatalf("error message leaks the password: %v", err)
			}
		})
	}
}
