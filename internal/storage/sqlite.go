package storage

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connection params are applied by the driver to every pooled connection;
// a one-off PRAGMA exec would only affect a single connection.
const sqliteParams = "_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"

// OpenSQLite opens a GORM connection to the SQLite database at path.
// Schema is managed by the SQL files in /migration, not AutoMigrate.
func OpenSQLite(path string, logLevel logger.LogLevel) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(withParams(path)), &gorm.Config{
		Logger:                 logger.Default.LogMode(logLevel),
		NowFunc:                func() time.Time { return time.Now().UTC() },
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		TranslateError:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}

	return db, nil
}

func withParams(path string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + sqliteParams
}
