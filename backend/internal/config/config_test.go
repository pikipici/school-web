package config_test

import (
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/config"
	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

func TestPublicDirDefaultsToPublicWhenAbsentOrBlank(t *testing.T) {
	for _, value := range []string{"", "   \t"} {
		t.Run(value, func(t *testing.T) {
			cfg, err := config.Load(func(key string) string {
				if key == "PUBLIC_DIR" {
					return value
				}
				return ""
			})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PublicDir != "public" {
				t.Fatalf("PublicDir = %q, want public", cfg.PublicDir)
			}
		})
	}
}

func TestPublicDirUsesTrimmedExplicitValue(t *testing.T) {
	cfg, err := config.Load(func(key string) string {
		if key == "PUBLIC_DIR" {
			return "  frontend/out  "
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicDir != "frontend/out" {
		t.Fatalf("PublicDir = %q", cfg.PublicDir)
	}
}

func TestAdminCookieSecureParsing(t *testing.T) {
	load := func(v string) (config.Config, error) {
		return config.Load(func(key string) string {
			if key == "ADMIN_COOKIE_SECURE" {
				return v
			}
			return ""
		})
	}

	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"", false}, {"  ", false}, {"false", false}, {"0", false},
		{"true", true}, {"TRUE", true}, {"1", true}, {" true ", true},
	} {
		cfg, err := load(tc.in)
		if err != nil {
			t.Fatalf("ADMIN_COOKIE_SECURE=%q: %v", tc.in, err)
		}
		if cfg.AdminCookieSecure != tc.want {
			t.Fatalf("ADMIN_COOKIE_SECURE=%q => %v; want %v", tc.in, cfg.AdminCookieSecure, tc.want)
		}
	}

	if _, err := load("maybe"); err == nil {
		t.Fatal("ADMIN_COOKIE_SECURE=maybe must be rejected, not silently treated as false")
	}
}

func TestCSPModeParsing(t *testing.T) {
	load := func(v string) (config.Config, error) {
		return config.Load(func(key string) string {
			if key == "CSP_MODE" {
				return v
			}
			return ""
		})
	}

	for in, want := range map[string]web.CSPMode{"": web.CSPEnforce, "enforce": web.CSPEnforce, "report-only": web.CSPReportOnly, "off": web.CSPOff} {
		cfg, err := load(in)
		if err != nil {
			t.Fatalf("CSP_MODE=%q: %v", in, err)
		}
		if cfg.CSPMode != want {
			t.Fatalf("CSP_MODE=%q => %v; want %v", in, cfg.CSPMode, want)
		}
	}
	if _, err := load("disabled"); err == nil {
		t.Fatal("CSP_MODE=disabled must be rejected, not silently treated as enforce or off")
	}
}

func TestAttachmentsDirIsOptionalAndTrimmed(t *testing.T) {
	load := func(v string) config.Config {
		cfg, err := config.Load(func(key string) string {
			if key == "ATTACHMENTS_DIR" {
				return v
			}
			return ""
		})
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	if got := load("").AttachmentsDir; got != "" {
		t.Errorf("unset ATTACHMENTS_DIR = %q, want empty (attachments disabled)", got)
	}
	if got := load("  /srv/data/attachments \t").AttachmentsDir; got != "/srv/data/attachments" {
		t.Errorf("ATTACHMENTS_DIR = %q", got)
	}
}
