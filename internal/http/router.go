package httpapi

import (
	"crypto/rsa"
	"net/http"
	"strings"

	authcontroller "github.com/andresramirez/auth-service/internal/controllers/auth"
	"github.com/andresramirez/auth-service/internal/services/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func NewRouter(controller *authcontroller.Controller, tokens *auth.TokenManager, publicKey *rsa.PublicKey, issuer string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/.well-known/jwks.json", func(c *gin.Context) { c.JSON(http.StatusOK, tokens.JWKS()) })

	api := router.Group("/v1/auth")
	api.POST("/register", controller.Register)
	api.POST("/login", controller.Login)
	api.POST("/refresh", controller.Refresh)
	api.POST("/logout", controller.Logout)
	api.GET("/me", authenticate(publicKey, issuer), controller.Me)
	return router
}

func authenticate(publicKey *rsa.PublicKey, issuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := authcontroller.ExtractBearerToken(c)
		if rawToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
			c.Abort()
			return
		}
		claims := &auth.AccessClaims{}
		token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return publicKey, nil
		}, jwt.WithIssuer(issuer))
		if err != nil || !token.Valid || claims.TokenType != "access" || strings.TrimSpace(claims.Subject) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			c.Abort()
			return
		}
		c.Set("user_id", claims.Subject)
		c.Next()
	}
}
