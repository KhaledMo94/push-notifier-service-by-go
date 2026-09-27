package models

import (
	"time"

	"gorm.io/gorm"
)

// StaleToken is a device token reported as invalid for a specific backend.
// DeletedAt is gorm.DeletedAt, so soft-deleted rows are excluded from every
// query by default; use db.Unscoped() to include them.
type StaleToken struct {
	ID        int64          `gorm:"primaryKey"                                          json:"id"`
	BackendID int64          `gorm:"not null;index;uniqueIndex:uq_stale_tokens_backend_token" json:"backend_id"`
	Token     string         `gorm:"not null;uniqueIndex:uq_stale_tokens_backend_token"       json:"token"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`

	Backend *Backend `gorm:"foreignKey:BackendID" json:"backend,omitempty"`
}

func (StaleToken) TableName() string { return "stale_tokens" }
