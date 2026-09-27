package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("fallbacks when unset", func(t *testing.T) {
		t.Setenv("DEFAULT_LOCALE", "")
		t.Setenv("DEFAULT_WORKER_COUNT", "")
		cfg, err := Load(filepath.Join(t.TempDir(), "missing.env"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DefaultLocale != fallbackLocale || cfg.DefaultWorkerCount != fallbackWorkerCount {
			t.Errorf("got %+v", cfg)
		}
	})

	t.Run("reads env file without overriding real env", func(t *testing.T) {
		t.Setenv("DEFAULT_LOCALE", "")
		t.Setenv("DEFAULT_WORKER_COUNT", "3")
		os.Unsetenv("DEFAULT_LOCALE")
		f := filepath.Join(t.TempDir(), ".env")
		if err := os.WriteFile(f, []byte("DEFAULT_LOCALE=ar\nDEFAULT_WORKER_COUNT=9\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DefaultLocale != "ar" || cfg.DefaultWorkerCount != 3 {
			t.Errorf("got %+v", cfg)
		}
	})

	for _, bad := range []string{"0", "-2", "abc"} {
		t.Run("invalid worker count "+bad, func(t *testing.T) {
			t.Setenv("DEFAULT_WORKER_COUNT", bad)
			if _, err := Load(filepath.Join(t.TempDir(), "missing.env")); err == nil {
				t.Errorf("expected error for %q", bad)
			}
		})
	}
}
