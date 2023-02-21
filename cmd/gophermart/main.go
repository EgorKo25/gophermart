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

	db, err := database.NewUserDB(cfg)
	if err != nil {
		log.Fatalf("%s", err)
	}

	cookie := cookies.NewCookieManager(cfg.SecretCookieKey, db)

	handler := handlers.NewHandler(db, cookie, cfg)

	myRouter := router.NewRouter(handler)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
