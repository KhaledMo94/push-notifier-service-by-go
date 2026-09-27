package repository

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrDuplicate         = errors.New("record already exists")
	ErrReferenceNotFound = errors.New("referenced record not found")
)

// mapError converts GORM errors into repository errors so callers
// don't depend on GORM.
func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return ErrDuplicate
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return ErrReferenceNotFound
	default:
		return err
	}
}
