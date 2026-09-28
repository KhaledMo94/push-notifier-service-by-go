package repository

import (
	"context"

	"github.com/KhaledMo94/push-notifier-service-by-go/internal/db"
	"github.com/KhaledMo94/push-notifier-service-by-go/internal/models"
)

// Consumers depend only on the role they need.

type BackendLoader interface {
	LoadAll(ctx context.Context) ([]models.Backend, error)
}

type BackendFinder interface {
	GetByID(ctx context.Context, id int64) (*models.Backend, error)
	GetByName(ctx context.Context, name string) (*models.Backend, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.Backend, error)
}

type BackendWriter interface {
	Create(ctx context.Context, b *models.Backend) error
	Update(ctx context.Context, b *models.Backend) error
	Delete(ctx context.Context, id int64) error
}

type BackendRepository struct {
	db *db.DB
}

// compile time error if not match all these interfcs
var (
	_ BackendLoader = (*BackendRepository)(nil)
	_ BackendFinder = (*BackendRepository)(nil)
	_ BackendWriter = (*BackendRepository)(nil)
)

func NewBackendRepository(d *db.DB) *BackendRepository {
	return &BackendRepository{db: d}
}

func (r *BackendRepository) Create(ctx context.Context, b *models.Backend) error {
	return mapError(r.db.WithContext(ctx).Create(b).Error)
}

func (r *BackendRepository) GetByID(ctx context.Context, id int64) (*models.Backend, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *BackendRepository) GetByName(ctx context.Context, name string) (*models.Backend, error) {
	return r.first(ctx, "name = ?", name)
}

func (r *BackendRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.Backend, error) {
	return r.first(ctx, "token_hash = ?", tokenHash)
}

// LoadAll returns every backend ordered by id; intended for warming in-memory state at startup.
func (r *BackendRepository) LoadAll(ctx context.Context) ([]models.Backend, error) {
	var backends []models.Backend
	if err := r.db.WithContext(ctx).Order("id").Find(&backends).Error; err != nil {
		return nil, mapError(err)
	}
	return backends, nil
}

// Update writes all editable columns, including zero values (e.g. clearing Description).
func (r *BackendRepository) Update(ctx context.Context, b *models.Backend) error {
	res := r.db.WithContext(ctx).
		Model(b).
		Select("Name", "Description", "TokenHash", "WorkerCount", "DefaultLocale", "FCMCredentials", "UpdatedAt").
		Updates(b)
	if res.Error != nil {
		return mapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete hard-deletes the backend; its stale tokens are removed by ON DELETE CASCADE.
func (r *BackendRepository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&models.Backend{}, id)
	if res.Error != nil {
		return mapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *BackendRepository) first(ctx context.Context, query string, arg any) (*models.Backend, error) {
	var b models.Backend
	if err := r.db.WithContext(ctx).Where(query, arg).First(&b).Error; err != nil {
		return nil, mapError(err)
	}
	return &b, nil
}
