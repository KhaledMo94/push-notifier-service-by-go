package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	fallbackLocale      = "en"
	fallbackWorkerCount = 1
)

type Config struct {
	DefaultLocale      string
	DefaultWorkerCount int
}

// Load reads configuration from the environment. Values in the given .env files
// are applied first without overriding variables already set in the environment;
// missing files are ignored.
func Load(envFiles ...string) (*Config, error) {
	if len(envFiles) == 0 {
		envFiles = []string{".env"}
	}
	for _, f := range envFiles {
		if err := godotenv.Load(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
	}

	workers, err := intEnv("DEFAULT_WORKER_COUNT", fallbackWorkerCount)
	if err != nil {
		return nil, err
	}
	if workers <= 0 {
		return nil, fmt.Errorf("DEFAULT_WORKER_COUNT must be > 0, got %d", workers)
	}

	return &Config{
		DefaultLocale:      stringEnv("DEFAULT_LOCALE", fallbackLocale),
		DefaultWorkerCount: workers,
	}, nil
}

func stringEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return n, nil
}
