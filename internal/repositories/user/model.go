package user

import (
	"time"

	"gorm.io/gorm"
)

type userRecord struct {
	ID           string `gorm:"primaryKey;type:uuid"`
	Email        string `gorm:"unique;not null"`
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userRecord) TableName() string { return "users" }

type refreshSessionRecord struct {
	ID        string     `gorm:"primaryKey;type:uuid"`
	UserID    string     `gorm:"index;type:uuid;not null"`
	TokenHash string     `gorm:"unique;not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
}

func (refreshSessionRecord) TableName() string { return "refresh_sessions" }

// Migrate preserves the existing schema while keeping GORM models in this layer.
// Unique fields use constraints so AutoMigrate preserves existing UNIQUE
// constraints, including those created with names other than GORM's defaults.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&userRecord{}, &refreshSessionRecord{})
}
