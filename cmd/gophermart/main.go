package main

import (
	"gophermart/internal/config"
	"gophermart/internal/database"
	"log"
)

func main() {

	config := config.NewConfig()

	db := database.NewUserDB(config)

	log.Println(db)
}
