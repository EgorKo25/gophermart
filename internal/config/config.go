package config

import (
	"flag"
)

type Config struct {
	Address string
	Port    string
	DB      string
}

func NewConfig() *Config {
	var cfg Config

	flag.StringVar(&cfg.Address,
		"a", "127.0.0.1",
		"Адрес, на котором располагается сервер",
	)
	flag.StringVar(&cfg.DB,
		"d", "postgresql://localhost:5432/postgres",
		"Адрес базы данных с которой работает сервер",
	)
	flag.StringVar(&cfg.Port,
		"p", "8080",
		"Порт на котором работает сервер",
	)
	flag.Parse()

	return &cfg
}
