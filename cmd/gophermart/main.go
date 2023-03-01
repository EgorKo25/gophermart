package main

import (
	"gophermart/internal/client"
	"log"
	"net/http"

	"gophermart/internal/config"
	"gophermart/internal/cookies"
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

	clientManager := client.NewClient(cfg, db)
	go clientManager.Run()

	cookie := cookies.NewCookieManager(cfg.SecretCookieKey, db)

	handler := handlers.NewHandler(db, cookie, cfg)

	myRouter := router.NewRouter(handler)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
