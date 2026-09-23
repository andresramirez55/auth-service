package database

import (
	"fmt"

	userrepository "github.com/andresramirez/auth-service/internal/repositories/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(url string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := userrepository.Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}
