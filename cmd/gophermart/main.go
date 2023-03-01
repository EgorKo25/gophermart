package main

import (
	"gophermart/internal/server/middleware"
	"log"
	"net/http"

	"gophermart/internal/client"
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

	middle := middleware.NewMiddleware(cookie)

	myRouter := router.NewRouter(handler, middle)

	log.Println(http.ListenAndServe(cfg.Address, myRouter))
}
