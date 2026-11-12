package adminctl_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/adminctl"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

const (
	goodPassword = "kopi-hangat-di-pagi-hari"
	newPassword  = "teh-manis-di-sore-hari"
)

type fixture struct {
	t      *testing.T
	pool   *pgxpool.Pool
	deps   adminctl.Deps
	hasher *password.Hasher
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testdb.OpenMigrated(t)
	hasher, err := password.NewHasher(password.Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{
		t: t, pool: pool, hasher: hasher,
		deps: adminctl.Deps{
			Users:    users.New(pool, nil),
			Sessions: session.New(pool, session.Options{}),
			Hasher:   hasher,
		},
	}
}

type result struct {
	code        int
	stdout, err string
}

// run executes a command with the password supplied on stdin.
func (f *fixture) run(stdin string, args ...string) result {
	f.t.Helper()
	var out, errOut bytes.Buffer
	code := adminctl.Run(context.Background(), args, adminctl.Streams{
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Stderr: &errOut,
	}, f.deps)
	r := result{code: code, stdout: out.String(), err: errOut.String()}
	f.assertNoSecrets(r, stdin)
	return r
}

// runInteractive executes a command as if on a terminal, answering prompts in order.
func (f *fixture) runInteractive(answers []string, args ...string) (result, []string) {
	f.t.Helper()
	var out, errOut bytes.Buffer
	var prompts []string
	next := 0
	code := adminctl.Run(context.Background(), args, adminctl.Streams{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &errOut,
		ReadSecret: func(prompt string) (string, error) {
			prompts = append(prompts, prompt)
			if next >= len(answers) {
				return "", errors.New("no more answers")
			}
			a := answers[next]
			next++
			return a, nil
		},
	}, f.deps)
	r := result{code: code, stdout: out.String(), err: errOut.String()}
	for _, a := range answers {
		f.assertNoSecrets(r, a)
	}
	return r, prompts
}

func (f *fixture) assertNoSecrets(r result, secret string) {
	f.t.Helper()
	s := strings.TrimSpace(secret)
	if s == "" {
		return
	}
	if strings.Contains(r.stdout, s) || strings.Contains(r.err, s) {
		f.t.Fatalf("command output leaks the password:\nstdout=%q\nstderr=%q", r.stdout, r.err)
	}
	if strings.Contains(r.stdout, "argon2") || strings.Contains(r.err, "argon2") {
		f.t.Fatal("command output leaks a password hash")
	}
}

func (f *fixture) ctx() context.Context {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	f.t.Cleanup(cancel)
	return c
}

func (f *fixture) userCount() int {
	f.t.Helper()
	var n int
	if err := f.pool.QueryRow(f.ctx(), "select count(*) from admin_users").Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) storedHash(username string) string {
	f.t.Helper()
	u, err := f.deps.Users.FindByUsername(f.ctx(), username)
	if err != nil {
		f.t.Fatalf("find %s: %v", username, err)
	}
	return u.PasswordHash
}

func (f *fixture) passwordWorks(username, pw string) bool {
	f.t.Helper()
	ok, _, err := f.hasher.Verify(pw, f.storedHash(username))
	if err != nil {
		f.t.Fatal(err)
	}
	return ok
}

func TestCreateAdminFromStdin(t *testing.T) {
	f := newFixture(t)

	r := f.run(goodPassword+"\n", "create-admin", "--username", "fikri", "--password-stdin")
	if r.code != 0 {
		t.Fatalf("exit = %d, stderr %q", r.code, r.err)
	}
	if f.userCount() != 1 || !f.passwordWorks("fikri", goodPassword) {
		t.Fatal("admin not created with the supplied password")
	}
	if !strings.Contains(r.stdout, "fikri") {
		t.Fatalf("stdout should confirm the username: %q", r.stdout)
	}
}

func TestStdinPasswordKeepsInnerSpacesAndStripsOnlyTheLineEnding(t *testing.T) {
	f := newFixture(t)
	pw := "  kopi hangat di pagi hari  "

	if r := f.run(pw+"\r\n", "create-admin", "--username", "fikri", "--password-stdin"); r.code != 0 {
		t.Fatalf("exit = %d, stderr %q", r.code, r.err)
	}
	if !f.passwordWorks("fikri", pw) {
		t.Fatal("password was trimmed or altered; only the trailing line ending may be removed")
	}
}

func TestCreateAdminRejectsInvalidUsername(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"ab", "has space", "semicolon;", strings.Repeat("a", 33), ""} {
		r := f.run(goodPassword+"\n", "create-admin", "--username", name, "--password-stdin")
		if r.code != 2 {
			t.Errorf("username %q: exit = %d; want 2", name, r.code)
		}
	}
	if f.userCount() != 0 {
		t.Fatal("a user was created with an invalid username")
	}
}

func TestUsernameIsNormalizedToLowerCase(t *testing.T) {
	f := newFixture(t)
	// Login lower-cases the username too, so " Fikri " and "fikri" are the same account.
	if r := f.run(goodPassword+"\n", "create-admin", "--username", " Fikri ", "--password-stdin"); r.code != 0 {
		t.Fatalf("exit = %d, stderr %q", r.code, r.err)
	}
	if _, err := f.deps.Users.FindByUsername(f.ctx(), "fikri"); err != nil {
		t.Fatalf("account not stored as lower-case fikri: %v", err)
	}
}

func TestCreateAdminEnforcesPasswordPolicy(t *testing.T) {
	f := newFixture(t)
	for name, pw := range map[string]string{
		"too short":         "Short1!",
		"common":            "password1234",
		"contains username": "my-fikri-secret-99",
		"low variety":       "aaaaaaaaaaaaaa",
		"empty":             "",
	} {
		r := f.run(pw+"\n", "create-admin", "--username", "fikri", "--password-stdin")
		if r.code != 2 {
			t.Errorf("%s: exit = %d; want 2", name, r.code)
		}
	}
	if f.userCount() != 0 {
		t.Fatal("a user was created with a weak password")
	}
}

func TestOnlyOneAdminCanBeCreated(t *testing.T) {
	f := newFixture(t)
	if r := f.run(goodPassword+"\n", "create-admin", "--username", "fikri", "--password-stdin"); r.code != 0 {
		t.Fatalf("first create failed: %q", r.err)
	}

	r := f.run(newPassword+"\n", "create-admin", "--username", "guru2", "--password-stdin")
	if r.code != 1 {
		t.Fatalf("second create exit = %d; want 1", r.code)
	}
	if f.userCount() != 1 {
		t.Fatal("a second admin was created")
	}
	if !f.passwordWorks("fikri", goodPassword) {
		t.Fatal("existing admin was modified by a rejected create")
	}
}

func TestResetPasswordReplacesHashAndRevokesSessions(t *testing.T) {
	f := newFixture(t)
	f.run(goodPassword+"\n", "create-admin", "--username", "fikri", "--password-stdin")
	u, _ := f.deps.Users.FindByUsername(f.ctx(), "fikri")
	iss, err := f.deps.Sessions.Create(f.ctx(), u.ID, "203.0.113.7", "test")
	if err != nil {
		t.Fatal(err)
	}

	r := f.run(newPassword+"\n", "reset-password", "--username", "fikri", "--password-stdin")
	if r.code != 0 {
		t.Fatalf("exit = %d, stderr %q", r.code, r.err)
	}
	if f.passwordWorks("fikri", goodPassword) {
		t.Fatal("old password still works after reset")
	}
	if !f.passwordWorks("fikri", newPassword) {
		t.Fatal("new password does not work after reset")
	}
	if _, err := f.deps.Sessions.Validate(f.ctx(), iss.Token); !errors.Is(err, session.ErrInvalid) {
		t.Fatalf("existing session survived a password reset: %v", err)
	}
}

func TestResetPasswordFailuresLeaveAccountUntouched(t *testing.T) {
	f := newFixture(t)
	f.run(goodPassword+"\n", "create-admin", "--username", "fikri", "--password-stdin")
	before := f.storedHash("fikri")

	if r := f.run(newPassword+"\n", "reset-password", "--username", "nobody", "--password-stdin"); r.code != 1 {
		t.Fatalf("unknown user exit = %d; want 1", r.code)
	}
	if r := f.run("short\n", "reset-password", "--username", "fikri", "--password-stdin"); r.code != 2 {
		t.Fatalf("weak password exit = %d; want 2", r.code)
	}
	if f.storedHash("fikri") != before {
		t.Fatal("hash changed although the reset was rejected")
	}
}

func TestInteractiveModePromptsTwiceAndRequiresMatch(t *testing.T) {
	f := newFixture(t)

	r, prompts := f.runInteractive([]string{goodPassword, "something-different-123"}, "create-admin", "--username", "fikri")
	if r.code != 2 {
		t.Fatalf("mismatched confirmation exit = %d; want 2", r.code)
	}
	if len(prompts) != 2 {
		t.Fatalf("prompts = %v; want a password prompt and a confirmation prompt", prompts)
	}
	if f.userCount() != 0 {
		t.Fatal("user created although confirmation did not match")
	}

	r, _ = f.runInteractive([]string{goodPassword, goodPassword}, "create-admin", "--username", "fikri")
	if r.code != 0 {
		t.Fatalf("matching confirmation exit = %d, stderr %q", r.code, r.err)
	}
	if !f.passwordWorks("fikri", goodPassword) {
		t.Fatal("interactive password not stored")
	}
}

func TestWithoutTerminalOrStdinFlagTheCommandRefuses(t *testing.T) {
	f := newFixture(t)
	r := f.run("", "create-admin", "--username", "fikri") // no --password-stdin, no ReadSecret
	if r.code != 2 {
		t.Fatalf("exit = %d; want 2", r.code)
	}
	if !strings.Contains(r.err, "--password-stdin") {
		t.Fatalf("error should explain the alternative: %q", r.err)
	}
}

func TestPasswordCannotBePassedAsArgument(t *testing.T) {
	f := newFixture(t)
	for _, flag := range []string{"--password", "-password", "--pass"} {
		r := f.run("", "create-admin", "--username", "fikri", flag, "kopi-hangat-di-pagi-hari")
		if r.code != 2 {
			t.Errorf("%s accepted (exit %d); secrets must not travel through argv", flag, r.code)
		}
	}
	if f.userCount() != 0 {
		t.Fatal("user created through a forbidden flag")
	}
}

func TestUsageErrors(t *testing.T) {
	f := newFixture(t)
	for name, args := range map[string][]string{
		"no command":      {},
		"unknown command": {"drop-everything"},
		"missing flag":    {"create-admin", "--password-stdin"},
	} {
		r := f.run("", args...)
		if r.code != 2 {
			t.Errorf("%s: exit = %d; want 2", name, r.code)
		}
		if r.stdout != "" {
			t.Errorf("%s: usage errors must go to stderr, got stdout %q", name, r.stdout)
		}
	}
	if r := f.run("", "help"); r.code != 0 || !strings.Contains(r.stdout, "create-admin") {
		t.Fatalf("help: exit %d, stdout %q", r.code, r.stdout)
	}
}

func TestOversizedStdinIsRejected(t *testing.T) {
	f := newFixture(t)
	r := f.run(strings.Repeat("a", 5000)+"\n", "create-admin", "--username", "fikri", "--password-stdin")
	if r.code != 2 {
		t.Fatalf("exit = %d; want 2", r.code)
	}
	if f.userCount() != 0 {
		t.Fatal("user created from oversized input")
	}
}
