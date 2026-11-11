package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/config"
	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
)

func main() {
	cfg, err := config.FromEnvironment()
	if err != nil {
		slog.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("API mulai", "address", cfg.Address, "environment", cfg.Environment)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("API berhenti", "error", err)
		os.Exit(1)
	}
}
