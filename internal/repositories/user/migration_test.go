package user

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Run against a disposable PostgreSQL database using TEST_DATABASE_URL.
func TestMigratePostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	for _, layout := range []string{"empty", "unique_constraints", "unique_indexes"} {
		t.Run(layout, func(t *testing.T) {
			db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			sqlDB.SetMaxOpenConns(1)
			schemaName := "migration_" + uuid.New().String()[:8]
			if err := db.Exec("CREATE SCHEMA " + schemaName).Error; err != nil {
				t.Fatal(err)
			}
			defer db.Exec("DROP SCHEMA " + schemaName + " CASCADE")
			if err := db.Exec("SET search_path TO " + schemaName).Error; err != nil {
				t.Fatal(err)
			}
			if layout != "empty" {
				for _, statement := range []string{
					`CREATE TABLE users (id uuid PRIMARY KEY, email text NOT NULL, password_hash text, name text, created_at timestamptz, updated_at timestamptz)`,
					`CREATE TABLE refresh_sessions (id uuid PRIMARY KEY, user_id uuid NOT NULL, token_hash text NOT NULL, expires_at timestamptz NOT NULL, revoked_at timestamptz, created_at timestamptz)`,
				} {
					if err := db.Exec(statement).Error; err != nil {
						t.Fatal(err)
					}
				}
				for _, spec := range []struct{ table, column string }{{"users", "email"}, {"refresh_sessions", "token_hash"}} {
					var statement string
					if layout == "unique_constraints" {
						statement = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT legacy_%s_%s UNIQUE (%s)", spec.table, spec.table, spec.column, spec.column)
					} else {
						statement = fmt.Sprintf("CREATE UNIQUE INDEX idx_%s_%s ON %s (%s)", spec.table, spec.column, spec.table, spec.column)
					}
					if err := db.Exec(statement).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			account := userRecord{ID: uuid.NewString(), Email: "existing@example.com", Name: "Existing"}
			session := refreshSessionRecord{ID: uuid.NewString(), UserID: account.ID, TokenHash: "existing-hash", ExpiresAt: time.Now().UTC().Add(time.Hour)}
			if layout != "empty" {
				if err := db.Create(&account).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&session).Error; err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 2; i++ {
				if err := Migrate(db); err != nil {
					t.Fatalf("migration %d: %v", i+1, err)
				}
			}
			if layout == "empty" {
				if err := db.Create(&account).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&session).Error; err != nil {
					t.Fatal(err)
				}
			}
			var stored userRecord
			if err := db.First(&stored, "id = ?", account.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Email != account.Email || stored.Name != account.Name {
				t.Fatal("existing user changed")
			}
			var storedSession refreshSessionRecord
			if err := db.First(&storedSession, "id = ?", session.ID).Error; err != nil {
				t.Fatal(err)
			}
			if storedSession.TokenHash != session.TokenHash || storedSession.UserID != account.ID {
				t.Fatal("existing session changed")
			}
			account.ID = uuid.NewString()
			if err := db.Create(&account).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
				t.Fatalf("expected duplicate email error, got %v", err)
			}
			session.ID = uuid.NewString()
			if err := db.Create(&session).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
				t.Fatalf("expected duplicate refresh hash error, got %v", err)
			}
		})
	}
}
