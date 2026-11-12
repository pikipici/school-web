// Command adminctl manages the admin account from the server's shell.
//
// Run it with the application's environment loaded so DATABASE_URL is set, e.g.
//
//	set -a; . /home/ubuntu/informatika-monitoring/.env; set +a
//	bin/adminctl create-admin --username fikri
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/pikipici/informatika-monitoring/backend/internal/adminctl"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
	"github.com/pikipici/informatika-monitoring/backend/internal/db"
)

func main() {
	os.Exit(run())
}

func run() int {
	streams := adminctl.Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		streams.ReadSecret = readSecret
	}

	args := os.Args[1:]
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Help and usage errors need no database.
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		return adminctl.Run(ctx, args, streams, adminctl.Deps{})
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL belum diatur; muat file .env aplikasi terlebih dahulu")
		return 1
	}

	openCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	pool, err := db.Open(openCtx, databaseURL)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tidak dapat terhubung ke database")
		return 1
	}
	defer pool.Close()

	hasher, err := password.NewHasher(password.DefaultParams())
	if err != nil {
		fmt.Fprintln(os.Stderr, "konfigurasi hash sandi tidak valid")
		return 1
	}

	return adminctl.Run(ctx, args, streams, adminctl.Deps{
		Users:    users.New(pool, nil),
		Sessions: session.New(pool, session.Options{}),
		Hasher:   hasher,
	})
}

// readSecret prompts on stderr and reads a line from the terminal without echo.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
