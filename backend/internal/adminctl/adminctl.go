// Package adminctl implements the server-side admin account commands
// (create-admin, reset-password). Passwords are never accepted as arguments or
// environment variables and are never printed.
package adminctl

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailure = 1 // the operation could not be completed
	exitUsage   = 2 // bad arguments or input; nothing was changed

	maxStdinBytes = 1024
)

// Streams are the process's I/O. ReadSecret reads a line without echo and is
// nil when there is no terminal.
type Streams struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	ReadSecret func(prompt string) (string, error)
}

// Deps are the services the commands act on.
type Deps struct {
	Users    *users.Store
	Sessions *session.Service
	Hasher   *password.Hasher
	// Audit is optional; when set, account changes are written to the audit log.
	Audit audit.Recorder
}

func (d Deps) record(ctx context.Context, e audit.Event) {
	if d.Audit != nil {
		if e.Detail == nil {
			e.Detail = map[string]string{}
		}
		e.Detail["via"] = "cli"
		d.Audit.Record(ctx, e)
	}
}

const usage = `adminctl - kelola akun admin di server

Perintah:
  create-admin    --username NAMA [--password-stdin]   Buat satu-satunya akun admin
  reset-password  --username NAMA [--password-stdin]   Ganti kata sandi dan akhiri semua sesi
  help                                                 Tampilkan bantuan ini

Kata sandi diminta lewat terminal (tanpa tampilan) atau dibaca dari stdin dengan
--password-stdin. Kata sandi tidak pernah diterima sebagai argumen atau variabel lingkungan.
`

// Run executes args (without the program name) and returns the exit code.
func Run(ctx context.Context, args []string, s Streams, d Deps) int {
	if len(args) == 0 {
		fmt.Fprint(s.Stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(s.Stdout, usage)
		return exitOK
	case "create-admin":
		return createAdmin(ctx, args[1:], s, d)
	case "reset-password":
		return resetPassword(ctx, args[1:], s, d)
	default:
		fmt.Fprintf(s.Stderr, "perintah tidak dikenal: %q\n\n%s", args[0], usage)
		return exitUsage
	}
}

type options struct {
	username      string
	passwordStdin bool
}

func parseOptions(name string, args []string, s Streams) (options, bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(s.Stderr)
	var o options
	fs.StringVar(&o.username, "username", "", "nama pengguna admin")
	fs.BoolVar(&o.passwordStdin, "password-stdin", false, "baca kata sandi dari stdin")
	if err := fs.Parse(args); err != nil {
		return options{}, false
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(s.Stderr, "argumen tidak dikenal: %q\n", fs.Arg(0))
		return options{}, false
	}
	o.username = strings.ToLower(strings.TrimSpace(o.username))
	return o, true
}

func createAdmin(ctx context.Context, args []string, s Streams, d Deps) int {
	o, ok := parseOptions("create-admin", args, s)
	if !ok {
		return exitUsage
	}
	if !users.ValidUsername(o.username) {
		fmt.Fprintln(s.Stderr, "nama pengguna tidak valid: 3-32 karakter dari a-z, 0-9, titik, garis bawah, atau tanda hubung")
		return exitUsage
	}

	pw, code := readNewPassword(o, s)
	if code != exitOK {
		return code
	}
	if err := password.ValidatePolicy(pw, o.username); err != nil {
		fmt.Fprintln(s.Stderr, policyMessage(err))
		return exitUsage
	}

	hash, err := d.Hasher.Hash(pw)
	if err != nil {
		fmt.Fprintln(s.Stderr, "gagal membuat hash kata sandi")
		return exitFailure
	}
	id, err := d.Users.CreateFirstAdmin(ctx, o.username, hash)
	if err != nil {
		if errors.Is(err, users.ErrAdminExists) {
			d.record(ctx, audit.Event{Action: "admin.created", Outcome: "failure", Detail: map[string]string{"username": o.username, "reason": "admin_exists"}})
			fmt.Fprintln(s.Stderr, "akun admin sudah ada; gunakan reset-password untuk mengganti kata sandi")
			return exitFailure
		}
		fmt.Fprintln(s.Stderr, "gagal menyimpan akun admin; periksa koneksi database")
		return exitFailure
	}
	d.record(ctx, audit.Event{Action: "admin.created", Outcome: "success", UserID: &id, Detail: map[string]string{"username": o.username}})
	fmt.Fprintf(s.Stdout, "Admin %q dibuat.\n", o.username)
	return exitOK
}

func resetPassword(ctx context.Context, args []string, s Streams, d Deps) int {
	o, ok := parseOptions("reset-password", args, s)
	if !ok {
		return exitUsage
	}
	if !users.ValidUsername(o.username) {
		fmt.Fprintln(s.Stderr, "nama pengguna tidak valid")
		return exitUsage
	}

	user, err := d.Users.FindByUsername(ctx, o.username)
	if errors.Is(err, users.ErrNotFound) {
		d.record(ctx, audit.Event{Action: "admin.password_reset", Outcome: "failure", Detail: map[string]string{"username": o.username, "reason": "not_found"}})
		fmt.Fprintln(s.Stderr, "akun tidak ditemukan")
		return exitFailure
	}
	if err != nil {
		fmt.Fprintln(s.Stderr, "gagal membaca akun; periksa koneksi database")
		return exitFailure
	}

	pw, code := readNewPassword(o, s)
	if code != exitOK {
		return code
	}
	if err := password.ValidatePolicy(pw, o.username); err != nil {
		fmt.Fprintln(s.Stderr, policyMessage(err))
		return exitUsage
	}

	hash, err := d.Hasher.Hash(pw)
	if err != nil {
		fmt.Fprintln(s.Stderr, "gagal membuat hash kata sandi")
		return exitFailure
	}
	if err := d.Users.SetPasswordHash(ctx, user.ID, hash); err != nil {
		fmt.Fprintln(s.Stderr, "gagal menyimpan kata sandi baru")
		return exitFailure
	}
	if err := d.Sessions.RevokeAllForUser(ctx, user.ID); err != nil {
		fmt.Fprintln(s.Stderr, "kata sandi sudah diganti, tetapi sesi lama GAGAL diakhiri; jalankan perintah ini lagi")
		return exitFailure
	}
	uid := user.ID
	d.record(ctx, audit.Event{Action: "admin.password_reset", Outcome: "success", UserID: &uid, Detail: map[string]string{"username": o.username}})
	fmt.Fprintf(s.Stdout, "Kata sandi %q diganti dan semua sesi diakhiri.\n", o.username)
	return exitOK
}

// readNewPassword gets a password from stdin or, on a terminal, prompts twice.
func readNewPassword(o options, s Streams) (string, int) {
	if o.passwordStdin {
		data, err := io.ReadAll(io.LimitReader(s.Stdin, maxStdinBytes+1))
		if err != nil {
			fmt.Fprintln(s.Stderr, "gagal membaca stdin")
			return "", exitFailure
		}
		if len(data) > maxStdinBytes {
			fmt.Fprintf(s.Stderr, "kata sandi terlalu panjang (maksimal %d byte)\n", password.MaxLength)
			return "", exitUsage
		}
		data = bytes.TrimSuffix(data, []byte("\n"))
		data = bytes.TrimSuffix(data, []byte("\r"))
		return string(data), exitOK
	}

	if s.ReadSecret == nil {
		fmt.Fprintln(s.Stderr, "tidak ada terminal; kirim kata sandi lewat stdin dengan --password-stdin")
		return "", exitUsage
	}
	first, err := s.ReadSecret("Kata sandi baru: ")
	if err != nil {
		fmt.Fprintln(s.Stderr, "gagal membaca kata sandi")
		return "", exitFailure
	}
	second, err := s.ReadSecret("Ulangi kata sandi: ")
	if err != nil {
		fmt.Fprintln(s.Stderr, "gagal membaca kata sandi")
		return "", exitFailure
	}
	if first != second {
		fmt.Fprintln(s.Stderr, "kedua kata sandi tidak sama")
		return "", exitUsage
	}
	return first, exitOK
}

func policyMessage(err error) string {
	switch {
	case errors.Is(err, password.ErrTooShort):
		return fmt.Sprintf("kata sandi terlalu pendek (minimal %d karakter)", password.MinLength)
	case errors.Is(err, password.ErrTooLong):
		return fmt.Sprintf("kata sandi terlalu panjang (maksimal %d byte)", password.MaxLength)
	case errors.Is(err, password.ErrContainsUsername):
		return "kata sandi tidak boleh memuat nama pengguna"
	case errors.Is(err, password.ErrCommon):
		return "kata sandi terlalu umum"
	case errors.Is(err, password.ErrLowVariety):
		return "kata sandi memakai terlalu sedikit karakter berbeda"
	default:
		return "kata sandi tidak memenuhi aturan"
	}
}
