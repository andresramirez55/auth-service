package user

import (
	"context"
	"errors"
)

var (
	ErrNotFound               = errors.New("record not found")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
)

// Repository defines the persistence operations required by authentication.
type Repository interface {
	Create(context.Context, *User) error
	FindByEmail(context.Context, string) (*User, error)
	FindByID(context.Context, string) (*User, error)
	CreateRefreshSession(context.Context, *RefreshSession) error
	ConsumeRefreshSession(context.Context, string) (*RefreshSession, error)
	RevokeRefreshSession(context.Context, string) error
}
