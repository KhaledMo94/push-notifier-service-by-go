package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
)

func TestStaleTokenRepository(t *testing.T) {
	ctx := context.Background()
	d := newTestDB(t)
	repo := NewStaleTokenRepository(d)

	backend := newBackend("shop", "hash-1")
	if err := NewBackendRepository(d).Create(ctx, backend); err != nil {
		t.Fatalf("create backend: %v", err)
	}

	n, err := repo.Record(ctx, backend.ID, []string{"a", "b", "a", "", "c"})
	if err != nil || n != 3 {
		t.Fatalf("record: n=%d err=%v, want 3", n, err)
	}
	if n, err = repo.Record(ctx, backend.ID, []string{"a", "d"}); err != nil || n != 1 {
		t.Fatalf("record with existing: n=%d err=%v, want 1", n, err)
	}
	if _, err := repo.Record(ctx, 999, []string{"x"}); !errors.Is(err, ErrReferenceNotFound) {
		t.Errorf("record unknown backend: got %v, want ErrReferenceNotFound", err)
	}

	stale, err := repo.FindStale(ctx, backend.ID, []string{"a", "c", "fresh"})
	sort.Strings(stale)
	if err != nil || fmt.Sprint(stale) != "[a c]" {
		t.Errorf("find stale: %v err=%v, want [a c]", stale, err)
	}

	page1, err := repo.ListByBackend(ctx, backend.ID, 0, 3)
	if err != nil || len(page1) != 3 {
		t.Fatalf("list page 1: len=%d err=%v", len(page1), err)
	}
	page2, err := repo.ListByBackend(ctx, backend.ID, page1[len(page1)-1].ID, 3)
	if err != nil || len(page2) != 1 || page2[0].Token != "d" {
		t.Fatalf("list page 2: %+v err=%v", page2, err)
	}

	if n, err = repo.MarkDeleted(ctx, backend.ID, []string{"a", "b", "missing"}); err != nil || n != 2 {
		t.Fatalf("mark deleted: n=%d err=%v, want 2", n, err)
	}
	if stale, _ = repo.FindStale(ctx, backend.ID, []string{"a", "b"}); len(stale) != 0 {
		t.Errorf("deleted tokens still reported stale: %v", stale)
	}
	dup := d.Create(&models.StaleToken{BackendID: backend.ID, Token: "b"}).Error
	if !errors.Is(mapError(dup), ErrDuplicate) {
		t.Errorf("insert duplicate of soft-deleted token: got %v, want ErrDuplicate", dup)
	}

	if n, err = repo.Record(ctx, backend.ID, []string{"a"}); err != nil || n != 1 {
		t.Fatalf("re-record deleted token: n=%d err=%v, want 1", n, err)
	}
	var rowsForA int64
	d.Unscoped().Model(&models.StaleToken{}).Where("backend_id = ? AND token = ?", backend.ID, "a").Count(&rowsForA)
	if rowsForA != 1 {
		t.Errorf("rows for revived token = %d, want 1", rowsForA)
	}
	if stale, _ = repo.FindStale(ctx, backend.ID, []string{"a"}); len(stale) != 1 {
		t.Errorf("revived token not reported stale: %v", stale)
	}

	if n, err = repo.Purge(ctx, time.Now().Add(-time.Hour)); err != nil || n != 0 {
		t.Errorf("purge before deletion: n=%d err=%v, want 0", n, err)
	}
	if n, err = repo.Purge(ctx, time.Now().Add(time.Second)); err != nil || n != 1 {
		t.Errorf("purge: n=%d err=%v, want 1 (b)", n, err)
	}
	var total int64
	d.Unscoped().Model(&models.StaleToken{}).Count(&total)
	if total != 3 {
		t.Errorf("rows after purge = %d, want 3 (a, c, d)", total)
	}
}

func TestStaleTokenRepositoryBatches(t *testing.T) {
	ctx := context.Background()
	d := newTestDB(t)
	repo := NewStaleTokenRepository(d)

	backend := newBackend("shop", "hash-1")
	if err := NewBackendRepository(d).Create(ctx, backend); err != nil {
		t.Fatalf("create backend: %v", err)
	}

	tokens := make([]string, staleTokenBatchSize*2+7)
	for i := range tokens {
		tokens[i] = fmt.Sprintf("tok-%d", i)
	}

	if n, err := repo.Record(ctx, backend.ID, tokens); err != nil || n != int64(len(tokens)) {
		t.Fatalf("record: n=%d err=%v, want %d", n, err, len(tokens))
	}
	if stale, err := repo.FindStale(ctx, backend.ID, tokens); err != nil || len(stale) != len(tokens) {
		t.Fatalf("find stale: len=%d err=%v", len(stale), err)
	}
	if n, err := repo.MarkDeleted(ctx, backend.ID, tokens); err != nil || n != int64(len(tokens)) {
		t.Fatalf("mark deleted: n=%d err=%v", n, err)
	}
}
