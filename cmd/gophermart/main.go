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
			err = client.OrdersUpdater()
			if err != nil {
				log.Printf("%s", err)
				return
			}
		}
	}()

	cookie := cookies.NewCookieManager(cfg.SecretCookieKey, db)

	handler := handlers.NewHandler(db, cookie, cfg)

	myRouter := router.NewRouter(handler, cookie)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
