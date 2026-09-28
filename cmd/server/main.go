package main

import (
	"log"
	"log/slog"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/config"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/crypto"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/logger"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	logger.Setup()

	cnf, err := config.Load()
	if err != nil {
		log.Fatalf("config loading failed : %v", err)
	}
	slog.Info("config loaded")

	dbConn, err := db.Open(cnf.DBPath, gormlogger.Warn)
	if err != nil {
		log.Fatalf("database connection failed : %v", err)
	}
	defer dbConn.Close()
	slog.Info("database connection established", "db", cnf.DBPath)

	cipher, err := crypto.NewAESGCM(cnf.EncKey)
	if err != nil {
		log.Fatalf("invalid ENCRYPTION_KEY: %v", err)
	}
	_ = cipher
}
