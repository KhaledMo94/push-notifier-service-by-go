package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"), logger.Silent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	files, err := filepath.Glob("../../migration/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)

	// raw sql.DB: GORM's prepared statements would run only the first statement of each file
	sqlDB, err := d.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		sqlText, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.Exec(string(sqlText)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	return d
}

func newBackend(name, tokenHash string) *models.Backend {
	return &models.Backend{
		Name:           name,
		TokenHash:      tokenHash,
		WorkerCount:    1,
		DefaultLocale:  "en",
		FCMCredentials: `{"type":"service_account"}`,
	}
}

func TestBackendRepository(t *testing.T) {
	ctx := context.Background()
	d := newTestDB(t)
	repo := NewBackendRepository(d)

	desc := "orders service"
	b := newBackend("shop", "hash-1")
	b.Description = &desc
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("create: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	got, err := repo.GetByTokenHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("get by token hash: %v", err)
	}
	if got.DefaultLocale != "en" || got.FCMCredentials != `{"type":"service_account"}` || got.WorkerCount != 1 {
		t.Errorf("fields not persisted: %+v", got)
	}

	if err := repo.Create(ctx, &models.Backend{Name: "shop", TokenHash: "hash-2"}); !errors.Is(err, ErrDuplicate) {
		t.Errorf("duplicate name: got %v, want ErrDuplicate", err)
	}
	if err := repo.Create(ctx, &models.Backend{Name: "other", TokenHash: "hash-3", WorkerCount: -1}); err == nil {
		t.Error("negative worker_count: expected CHECK constraint error")
	}

	got.Description = nil
	got.WorkerCount = 8
	got.DefaultLocale = "ar"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, err := repo.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if updated.Description != nil || updated.WorkerCount != 8 || updated.DefaultLocale != "ar" {
		t.Errorf("update not applied: %+v", updated)
	}
	if !updated.UpdatedAt.After(b.UpdatedAt) {
		t.Error("expected updated_at to advance")
	}

	missing := newBackend("x", "y")
	missing.ID = 999
	if err := repo.Update(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("update missing: got %v, want ErrNotFound", err)
	}

	list, err := repo.LoadAll(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("load all: len=%d err=%v", len(list), err)
	}

	if err := d.Create(&models.StaleToken{BackendID: b.ID, Token: "dev-1"}).Error; err != nil {
		t.Fatalf("create stale token: %v", err)
	}
	if err := repo.Delete(ctx, b.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var remaining int64
	d.Unscoped().Model(&models.StaleToken{}).Count(&remaining)
	if remaining != 0 {
		t.Errorf("stale tokens after cascade = %d, want 0", remaining)
	}

	if _, err := repo.GetByName(ctx, "shop"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get deleted: got %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing: got %v, want ErrNotFound", err)
	}
}
