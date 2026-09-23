package main

import (
	"log"
	"time"

	"github.com/andresramirez/auth-service/internal/config"
	authcontroller "github.com/andresramirez/auth-service/internal/controllers/auth"
	"github.com/andresramirez/auth-service/internal/database"
	httpapi "github.com/andresramirez/auth-service/internal/http"
	userrepository "github.com/andresramirez/auth-service/internal/repositories/user"
	"github.com/andresramirez/auth-service/internal/services/auth"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	tokens, err := auth.NewTokenManager(cfg.PrivateKeyPEM, cfg.Issuer, time.Duration(cfg.AccessTokenTTLMin)*time.Minute)
	if err != nil {
		log.Fatal(err)
	}
	users := userrepository.New(db)
	service := auth.NewService(users, tokens, time.Duration(cfg.RefreshTokenTTLDays)*24*time.Hour)
	controller := authcontroller.New(service)
	router := httpapi.NewRouter(controller, tokens, tokens.PublicKey(), cfg.Issuer)

	log.Printf("auth service listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
