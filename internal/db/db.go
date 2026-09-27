package db

import (
	"fmt"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB embeds *gorm.DB so every GORM method is available directly,
// while allowing project-specific query methods to be defined on it.
type DB struct {
	*gorm.DB
}

func New(conn *gorm.DB) *DB {
	return &DB{DB: conn}
}

func Open(path string, logLevel logger.LogLevel) (*DB, error) {
	conn, err := storage.OpenSQLite(path, logLevel)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	return sqlDB.Close()
}
