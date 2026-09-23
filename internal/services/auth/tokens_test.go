package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAccessTokenCreatesVerifiableRS256Token(t *testing.T) {
	manager, err := NewTokenManager("", "https://auth.example.test", time.Minute)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	rawToken, expiresAt, err := manager.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("access token should expire in the future")
	}

	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		return manager.PublicKey(), nil
	}, jwt.WithIssuer("https://auth.example.test"))
	if err != nil || !parsed.Valid {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.Subject != "user-123" || claims.TokenType != "access" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
