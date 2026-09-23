package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/andresramirez/auth-service/internal/domain/user"
)

// Embedding the contract makes unexpected persistence calls fail the test.
type userRepositoryStub struct {
	user.Repository
	create      func(context.Context, *user.User) error
	findByEmail func(context.Context, string) (*user.User, error)
	consume     func(context.Context, string) (*user.RefreshSession, error)
}

func (r userRepositoryStub) Create(ctx context.Context, account *user.User) error {
	return r.create(ctx, account)
}

func (r userRepositoryStub) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.findByEmail(ctx, email)
}

func (r userRepositoryStub) ConsumeRefreshSession(ctx context.Context, hash string) (*user.RefreshSession, error) {
	return r.consume(ctx, hash)
}

func TestRegisterNormalizesAccountAndHandlesDuplicateEmail(t *testing.T) {
	repository := userRepositoryStub{create: func(_ context.Context, account *user.User) error {
		if account.Email != "ana@example.com" || account.Name != "Ana" {
			t.Fatalf("unexpected account: email=%q name=%q", account.Email, account.Name)
		}
		return user.ErrEmailAlreadyRegistered
	}}
	service := NewService(repository, nil, 0)
	_, _, err := service.Register(context.Background(), RegisterInput{
		Email: " Ana@Example.COM ", Password: "una-clave-segura", Name: " Ana ",
	})
	if err == nil || err.Error() != "email already registered" {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
}

func TestLoginNormalizesEmailBeforeLookup(t *testing.T) {
	repository := userRepositoryStub{findByEmail: func(_ context.Context, email string) (*user.User, error) {
		if email != "ana@example.com" {
			t.Fatalf("unexpected email: %q", email)
		}
		return nil, user.ErrNotFound
	}}
	_, _, err := NewService(repository, nil, 0).Login(context.Background(), LoginInput{Email: " Ana@Example.COM "})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRefreshTranslatesDomainNotFound(t *testing.T) {
	repository := userRepositoryStub{consume: func(_ context.Context, hash string) (*user.RefreshSession, error) {
		if hash != HashRefreshToken("expired-token") {
			t.Fatal("expected a hashed refresh token")
		}
		return nil, user.ErrNotFound
	}}
	_, err := NewService(repository, nil, 0).Refresh(context.Background(), "expired-token")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected invalid refresh token, got %v", err)
	}
}
