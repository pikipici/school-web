package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Address     string
	DatabaseURL string
	PublicDir   string
	Environment string
}

func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Address:     valueOrDefault(getenv("APP_ADDR"), ":8300"),
		DatabaseURL: strings.TrimSpace(getenv("DATABASE_URL")),
		PublicDir:   valueOrDefault(getenv("PUBLIC_DIR"), "public"),
		Environment: valueOrDefault(getenv("APP_ENV"), "development"),
	}
	if !strings.HasPrefix(cfg.Address, ":") {
		return Config{}, fmt.Errorf("APP_ADDR tidak valid")
	}
	return cfg, nil
}

func FromEnvironment() (Config, error) {
	return Load(os.Getenv)
}

func valueOrDefault(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
