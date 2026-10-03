package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/crypto"
	"github.com/joho/godotenv"
)

const (
	fallbackLocale      = "en"
	fallbackWorkerCount = 1
	fallbackAlgorithm   = crypto.AlgorithmAESGCM
)

// encryptionKeyEnv maps each supported algorithm to the variable holding its key.
var encryptionKeyEnv = map[string]string{
	crypto.AlgorithmAESGCM:           "AES_GCM_KEY",
	crypto.AlgorithmChaCha20Poly1305: "CHACHA20_POLY1305_KEY",
}

type Config struct {
	DBPath             string
	DefaultLocale      string
	DefaultWorkerCount int
	Encryption         EncryptionConfig
}

type EncryptionConfig struct {
	Algorithm string
	Key       string // key of the selected algorithm only
}

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

	encryption, err := loadEncryption()
	if err != nil {
		return nil, err
	}

	return &Config{
		DBPath:             stringEnv("DB_PATH", "notifier.db"),
		DefaultLocale:      stringEnv("DEFAULT_LOCALE", fallbackLocale),
		DefaultWorkerCount: workers,
		Encryption:         encryption,
	}, nil
}

func loadEncryption() (EncryptionConfig, error) {
	algorithm := strings.ToLower(stringEnv("ENCRYPTION_ALGORITHM", fallbackAlgorithm))

	keyEnv, ok := encryptionKeyEnv[algorithm]
	if !ok {
		return EncryptionConfig{}, fmt.Errorf("ENCRYPTION_ALGORITHM must be %q or %q, got %q",
			crypto.AlgorithmAESGCM, crypto.AlgorithmChaCha20Poly1305, algorithm)
	}

	key := strings.TrimSpace(os.Getenv(keyEnv))
	if key == "" {
		return EncryptionConfig{}, fmt.Errorf("%s is required when ENCRYPTION_ALGORITHM=%s", keyEnv, algorithm)
	}

	return EncryptionConfig{Algorithm: algorithm, Key: key}, nil
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
