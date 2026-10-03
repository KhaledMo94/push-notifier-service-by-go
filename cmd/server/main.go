package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/crypto"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/logger"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	logger.Setup()

	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// run holds the startup logic; returning an error (instead of exiting) lets deferred cleanup run.
func run() error {
	cnf, err := config.Load()
	if err != nil {
		return fmt.Errorf("config loading failed: %w", err)
	}
	slog.Info("config loaded")

	dbConn, err := db.Open(cnf.DBPath, gormlogger.Warn)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer dbConn.Close()
	slog.Info("database connection established", "db", cnf.DBPath)

	var cipher crypto.Cipher
	cipher, err = crypto.NewCipher(cnf.Encryption.Algorithm, cnf.Encryption.Key)
	if err != nil {
		return fmt.Errorf("encryption setup failed: %w", err)
	}
	slog.Info("encryption ready", "algorithm", cnf.Encryption.Algorithm)
	_ = cipher

	return nil
}
