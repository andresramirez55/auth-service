package user

import (
	"context"
	"errors"
	"time"

	"github.com/andresramirez/auth-service/internal/domain/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ user.Repository = (*Repository)(nil)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, account *user.User) error {
	record := userRecord(*account)
	record.ID = uuid.NewString()
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user.ErrEmailAlreadyRegistered
		}
		return err
	}
	*account = user.User(record)
	return nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var record userRecord
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrNotFound
	}
	account := user.User(record)
	return &account, err
}

func (r *Repository) FindByID(ctx context.Context, id string) (*user.User, error) {
	var record userRecord
	err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrNotFound
	}
	account := user.User(record)
	return &account, err
}

func (r *Repository) CreateRefreshSession(ctx context.Context, session *user.RefreshSession) error {
	record := refreshSessionRecord(*session)
	record.ID = uuid.NewString()
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	*session = user.RefreshSession(record)
	return nil
}

func (r *Repository) ConsumeRefreshSession(ctx context.Context, hash string) (*user.RefreshSession, error) {
	var session refreshSessionRecord
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&session).Update("revoked_at", now).Error; err != nil {
		return nil, err
	}
	result := user.RefreshSession(session)
	return &result, nil
}

func (r *Repository) RevokeRefreshSession(ctx context.Context, hash string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&refreshSessionRecord{}).Where("token_hash = ? AND revoked_at IS NULL", hash).Update("revoked_at", now).Error
}
