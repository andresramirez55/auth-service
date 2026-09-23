package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andresramirez/auth-service/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type RegisterInput struct{ Email, Password, Name string }
type LoginInput struct{ Email, Password string }

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Service struct {
	users      user.Repository
	tokens     *TokenManager
	refreshTTL time.Duration
}

func NewService(users user.Repository, tokens *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{users: users, tokens: tokens, refreshTTL: refreshTTL}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*user.User, *TokenPair, error) {
	if err := validateCredentials(input.Email, input.Password); err != nil {
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}
	created := &user.User{Email: strings.ToLower(strings.TrimSpace(input.Email)), Name: strings.TrimSpace(input.Name), PasswordHash: string(hash)}
	if err := s.users.Create(ctx, created); err != nil {
		if errors.Is(err, user.ErrEmailAlreadyRegistered) {
			return nil, nil, fmt.Errorf("email already registered")
		}
		return nil, nil, fmt.Errorf("create user: %w", err)
	}
	pair, err := s.issueTokenPair(ctx, created.ID)
	return created, pair, err
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*user.User, *TokenPair, error) {
	account, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(input.Password)) != nil {
		return nil, nil, ErrInvalidCredentials
	}
	pair, err := s.issueTokenPair(ctx, account.ID)
	return account, pair, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	session, err := s.users.ConsumeRefreshSession(ctx, HashRefreshToken(refreshToken))
	if errors.Is(err, user.ErrNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, session.UserID)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.users.RevokeRefreshSession(ctx, HashRefreshToken(refreshToken))
}

func (s *Service) GetUser(ctx context.Context, id string) (*user.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *Service) issueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	accessToken, expiresAt, err := s.tokens.IssueAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken := s.tokens.NewRefreshToken()
	if err := s.users.CreateRefreshSession(ctx, &user.RefreshSession{UserID: userID, TokenHash: HashRefreshToken(refreshToken), ExpiresAt: time.Now().UTC().Add(s.refreshTTL)}); err != nil {
		return nil, fmt.Errorf("store refresh session: %w", err)
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresAt: expiresAt}, nil
}

func validateCredentials(email, password string) error {
	if strings.TrimSpace(email) == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("a valid email is required")
	}
	if len(password) < 12 {
		return fmt.Errorf("password must have at least 12 characters")
	}
	return nil
}
