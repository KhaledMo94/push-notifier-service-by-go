package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migration/*.sql
var migrationFiles embed.FS

func(d *DB) Migrate() (uint,error){
	sqlDB,err := d.DB.DB()
	if err!=nil{
		return 0, fmt.Errorf("get sql.DB: %w", err)
	}

	driver,err :=sqlite3.WithInstance(sqlDB,&sqlite3.Config{})
	if err != nil {
		return 0, fmt.Errorf("migration driver: %w", err)
	}

	src, err := iofs.New(migrationFiles, "migration")
	if err != nil {
		return 0, fmt.Errorf("migration source: %w", err)
	}

	m,err := migrate.NewWithInstance("iofs",src,"sqlite3",driver)
	if err != nil {
		return 0, fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return 0, fmt.Errorf("migration version: %w", err)
	}
	if dirty {
		return version, fmt.Errorf("schema version %d is dirty", version)
	}
	return version, nil
}