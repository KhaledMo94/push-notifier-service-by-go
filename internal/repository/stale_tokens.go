package repository

import (
	"context"
	"time"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	staleTokenBatchSize   = 500
	staleTokenMaxPageSize = 1000
)

// Consumers depend only on the role they need.

type StaleTokenRecorder interface {
	Record(ctx context.Context, backendID int64, tokens []string) (int64, error)
}

type StaleTokenReader interface {
	ListByBackend(ctx context.Context, backendID, afterID int64, limit int) ([]models.StaleToken, error)
	FindStale(ctx context.Context, backendID int64, tokens []string) ([]string, error)
}

type StaleTokenRemover interface {
	MarkDeleted(ctx context.Context, backendID int64, tokens []string) (int64, error)
	Purge(ctx context.Context, deletedBefore time.Time) (int64, error)
}

type StaleTokenRepository struct {
	db *db.DB
}

var (
	_ StaleTokenRecorder = (*StaleTokenRepository)(nil)
	_ StaleTokenReader   = (*StaleTokenRepository)(nil)
	_ StaleTokenRemover  = (*StaleTokenRepository)(nil)
)

func NewStaleTokenRepository(d *db.DB) *StaleTokenRepository {
	return &StaleTokenRepository{db: d}
}

// classic deleted at conflict (revive it to notify the other backend to delete)
var reviveOnConflict = clause.OnConflict{
	Columns: []clause.Column{{Name: "backend_id"}, {Name: "token"}},
	DoUpdates: clause.Assignments(map[string]any{
		"deleted_at": nil,
		"created_at": gorm.Expr("excluded.created_at"),
	}),
	Where: clause.Where{Exprs: []clause.Expression{
		clause.Expr{SQL: "stale_tokens.deleted_at IS NOT NULL"},
	}},
}

func (r *StaleTokenRepository) Record(ctx context.Context, backendID int64, tokens []string) (int64, error) {
	var recorded int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, batch := range chunk(uniqueNonEmpty(tokens), staleTokenBatchSize) {
			rows := make([]models.StaleToken, len(batch))
			for i, token := range batch {
				rows[i] = models.StaleToken{BackendID: backendID, Token: token}
			}
			res := tx.Clauses(reviveOnConflict).Create(&rows)
			if res.Error != nil {
				return res.Error
			}
			recorded += res.RowsAffected
		}
		return nil
	})
	if err != nil {
		return 0, mapError(err)
	}
	return recorded, nil
}

// ListByBackend pages through active stale tokens using keyset pagination:
// pass the last returned ID as afterID (0 for the first page).
func (r *StaleTokenRepository) ListByBackend(ctx context.Context, backendID, afterID int64, limit int) ([]models.StaleToken, error) {
	if limit <= 0 || limit > staleTokenMaxPageSize {
		limit = staleTokenMaxPageSize
	}
	var tokens []models.StaleToken
	err := r.db.WithContext(ctx).
		Where("backend_id = ? AND id > ?", backendID, afterID).
		Order("id").
		Limit(limit).
		Find(&tokens).Error
	if err != nil {
		return nil, mapError(err)
	}
	return tokens, nil
}

// FindStale returns the subset of tokens that are currently stale for the backend.
func (r *StaleTokenRepository) FindStale(ctx context.Context, backendID int64, tokens []string) ([]string, error) {
	var stale []string
	for _, batch := range chunk(uniqueNonEmpty(tokens), staleTokenBatchSize) {
		var found []string
		err := r.db.WithContext(ctx).
			Model(&models.StaleToken{}).
			Where("backend_id = ? AND token IN ?", backendID, batch).
			Pluck("token", &found).Error
		if err != nil {
			return nil, mapError(err)
		}
		stale = append(stale, found...)
	}
	return stale, nil
}

// MarkDeleted soft-deletes the backend's stale tokens (e.g. after the backend
// has cleaned them up). Returns the number of tokens marked.
func (r *StaleTokenRepository) MarkDeleted(ctx context.Context, backendID int64, tokens []string) (int64, error) {
	var deleted int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, batch := range chunk(uniqueNonEmpty(tokens), staleTokenBatchSize) {
			res := tx.Where("backend_id = ? AND token IN ?", backendID, batch).Delete(&models.StaleToken{})
			if res.Error != nil {
				return res.Error
			}
			deleted += res.RowsAffected
		}
		return nil
	})
	if err != nil {
		return 0, mapError(err)
	}
	return deleted, nil
}

// Purge hard-deletes tokens soft-deleted before the given time.
func (r *StaleTokenRepository) Purge(ctx context.Context, deletedBefore time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", deletedBefore.UTC()).
		Delete(&models.StaleToken{})
	if res.Error != nil {
		return 0, mapError(res.Error)
	}
	return res.RowsAffected, nil
}
