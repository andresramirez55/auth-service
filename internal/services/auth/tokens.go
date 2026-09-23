package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	privateKey *rsa.PrivateKey
	issuer     string
	kid        string
	accessTTL  time.Duration
}

func NewTokenManager(privateKeyPEM, issuer string, accessTTL time.Duration) (*TokenManager, error) {
	key, err := loadPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	hash := sha256.Sum256(publicDER)
	return &TokenManager{privateKey: key, issuer: issuer, kid: base64.RawURLEncoding.EncodeToString(hash[:8]), accessTTL: accessTTL}, nil
}

func (m *TokenManager) IssueAccessToken(userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessTTL)
	claims := AccessClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: m.issuer, Subject: userID, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt), ID: randomToken(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.kid
	signed, err := token.SignedString(m.privateKey)
	return signed, expiresAt, err
}

func (m *TokenManager) NewRefreshToken() string { return randomToken() }

func (m *TokenManager) PublicKey() *rsa.PublicKey { return &m.privateKey.PublicKey }

func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func (m *TokenManager) JWKS() map[string]any {
	public := m.privateKey.PublicKey
	return map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": m.kid,
		"n": base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(public.E)).Bytes()),
	}}}
}

func loadPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	if privateKeyPEM == "" {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate development key: %w", err)
		}
		return key, nil
	}
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("AUTH_PRIVATE_KEY_PEM is not valid PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	rsaKey, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if pkcs1Err == nil {
		return rsaKey, nil
	}
	return nil, fmt.Errorf("AUTH_PRIVATE_KEY_PEM must contain an RSA private key")
}

func randomToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to read secure random bytes")
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
