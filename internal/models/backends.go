package models

import "time"

// Backend is a service allowed to send push notifications through the notifier.
type Backend struct {
	ID             int64     `gorm:"primaryKey"      json:"id"`
	Name           string    `gorm:"not null;unique" json:"name"`
	Description    *string   `json:"description,omitempty"`
	TokenHash      string    `gorm:"not null;unique" json:"-"`
	WorkerCount    int       `gorm:"not null"        json:"worker_count"`
	DefaultLocale  string    `gorm:"not null"        json:"default_locale"`
	FCMCredentials string    `gorm:"column:fcm_credentials;not null" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	StaleTokens []StaleToken `gorm:"foreignKey:BackendID;constraint:OnDelete:CASCADE" json:"stale_tokens,omitempty"`
}

func (Backend) TableName() string { return "backends" }
