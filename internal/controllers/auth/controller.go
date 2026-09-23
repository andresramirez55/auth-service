package authcontroller

import (
	"net/http"
	"strings"

	"github.com/andresramirez/auth-service/internal/services/auth"
	"github.com/gin-gonic/gin"
)

type Controller struct{ service *auth.Service }

func New(service *auth.Service) *Controller { return &Controller{service: service} }

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (ctrl *Controller) Register(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	account, tokens, err := ctrl.service.Register(c.Request.Context(), auth.RegisterInput{Email: request.Email, Password: request.Password, Name: request.Name})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": account, "tokens": tokens})
}

func (ctrl *Controller) Login(c *gin.Context) {
	var request credentialsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	account, tokens, err := ctrl.service.Login(c.Request.Context(), auth.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": account, "tokens": tokens})
}

func (ctrl *Controller) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tokens, err := ctrl.service.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

func (ctrl *Controller) Logout(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := ctrl.service.Logout(c.Request.Context(), request.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close session"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *Controller) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	account, err := ctrl.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": account})
}

func ExtractBearerToken(c *gin.Context) string {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
