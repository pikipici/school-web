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

	"github.com/pikipici/informatika-monitoring/backend/internal/config"
	"github.com/pikipici/informatika-monitoring/backend/internal/db"
	"github.com/pikipici/informatika-monitoring/backend/internal/health"
	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
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
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		pool, err := db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		pinger = pool
	}

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouterWithPinger(pinger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API mulai", "address", cfg.Address, "environment", cfg.Environment, "database_configured", pinger != nil)
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
