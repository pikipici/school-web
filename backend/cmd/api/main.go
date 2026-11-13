package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/ratelimit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
	"github.com/pikipici/informatika-monitoring/backend/internal/config"
	"github.com/pikipici/informatika-monitoring/backend/internal/content"
	"github.com/pikipici/informatika-monitoring/backend/internal/db"
	"github.com/pikipici/informatika-monitoring/backend/internal/health"
	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("API berhenti", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnvironment()
	if err != nil {
		return fmt.Errorf("konfigurasi tidak valid: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var pinger health.Pinger
	var admin *httpapi.AdminAuth
	var publicContent content.PublicReader
	var adminContent content.AdminStore
	var auditStore *audit.Store
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		pool, err := db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		pinger = pool
		contentStore := content.New(pool, nil)
		publicContent, adminContent = contentStore, contentStore
		auditStore = audit.New(pool, nil)

		hasher, err := password.NewHasher(password.DefaultParams())
		if err != nil {
			return fmt.Errorf("konfigurasi hash sandi tidak valid: %w", err)
		}
		sessions := session.New(pool, session.Options{})
		admin = httpapi.NewAdminAuth(httpapi.AdminAuthConfig{
			Users:        users.New(pool, nil),
			Sessions:     sessions,
			Limiter:      ratelimit.New(ratelimit.Options{}),
			Hasher:       hasher,
			Gate:         ratelimit.NewGate(2),
			SecureCookie: cfg.AdminCookieSecure,
			Audit:        auditStore,
		})
		go purgeSessions(ctx, sessions)
	}

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouterWithDeps(httpapi.Deps{Pinger: pinger, PublicDir: cfg.PublicDir, Admin: admin, CSPMode: cfg.CSPMode, PublicContent: publicContent, AdminContent: adminContent, ContentAudit: auditStore}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API mulai", "address", cfg.Address, "environment", cfg.Environment, "database_configured", pinger != nil, "admin_auth_enabled", admin != nil, "cookie_secure", cfg.AdminCookieSecure, "csp_mode", cspModeName(cfg.CSPMode))
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			shutdownErr := err
			_ = server.Close()
			<-serverErrors
			return fmt.Errorf("graceful shutdown gagal: %w", shutdownErr)
		}
		if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server berhenti saat shutdown: %w", err)
		}
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server gagal: %w", err)
	}
}

// purgeSessions removes expired admin sessions every ten minutes until ctx ends.
func purgeSessions(ctx context.Context, sessions *session.Service) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purgeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if n, err := sessions.PurgeExpired(purgeCtx); err != nil {
				slog.Error("pembersihan sesi gagal", "error", err)
			} else if n > 0 {
				slog.Info("sesi kedaluwarsa dihapus", "count", n)
			}
			cancel()
		}
	}
}

func cspModeName(m web.CSPMode) string {
	switch m {
	case web.CSPReportOnly:
		return "report-only"
	case web.CSPOff:
		return "off"
	default:
		return "enforce"
	}
}
