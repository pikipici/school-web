package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

type Config struct {
	Address     string
	DatabaseURL string
	PublicDir   string
	Environment string
	// AdminCookieSecure must be true when the site is served over HTTPS.
	AdminCookieSecure bool
	// CSPMode controls the Content-Security-Policy header on HTML pages.
	CSPMode web.CSPMode
	// AttachmentsDir holds uploaded material files. Empty disables attachments.
	AttachmentsDir string
}

func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Address:        valueOrDefault(getenv("APP_ADDR"), ":8300"),
		DatabaseURL:    strings.TrimSpace(getenv("DATABASE_URL")),
		PublicDir:      valueOrDefault(getenv("PUBLIC_DIR"), "public"),
		Environment:    valueOrDefault(getenv("APP_ENV"), "development"),
		AttachmentsDir: strings.TrimSpace(getenv("ATTACHMENTS_DIR")),
	}
	secure, err := parseBool(getenv("ADMIN_COOKIE_SECURE"))
	if err != nil {
		return Config{}, fmt.Errorf("ADMIN_COOKIE_SECURE tidak valid")
	}
	cfg.AdminCookieSecure = secure
	mode, err := web.ParseCSPMode(getenv("CSP_MODE"))
	if err != nil {
		return Config{}, fmt.Errorf("CSP_MODE tidak valid")
	}
	cfg.CSPMode = mode
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

// parseBool accepts true/false/1/0 (any case); blank means false. Anything else is an error so a typo cannot silently weaken cookie security.
func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "false", "0":
		return false, nil
	case "true", "1":
		return true, nil
	default:
		return false, fmt.Errorf("invalid boolean")
	}
}
