package config_test

import (
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/config"
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
