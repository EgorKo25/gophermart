package main

import (
	client2 "gophermart/internal/client"
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

	client := client2.NewClient(cfg, db)
	go func() {
		for {
			client.OrdersUpdater()
		}
	}()

	cookie := cookies.NewCookieManager(cfg.SecretCookieKey, db)

	handler := handlers.NewHandler(db, cookie, cfg)

	myRouter := router.NewRouter(handler)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
