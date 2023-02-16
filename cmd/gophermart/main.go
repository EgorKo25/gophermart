package main

import (
	"gophermart/internal/cookies"
	"log"
	"net/http"

	"gophermart/internal/config"
	"gophermart/internal/database"
	"gophermart/internal/server/handlers"
	"gophermart/internal/server/router"
)

func main() {

	cfg := config.NewConfig()

	db := database.NewUserDB(cfg)

	coockieFormat := cookies.NewCoockieFormat(cfg.SecretCookieKey)

	handler := handlers.NewHandler(db, coockieFormat)

	myRouter := router.NewRouter(handler)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
