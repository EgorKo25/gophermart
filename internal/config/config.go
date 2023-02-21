package config

import (
	"encoding/hex"
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address         string `env:"RUN_ADDRESS"`
	DB              string `env:"DATABASE_URI"`
	SecretCookieKey []byte
	BlackBox        string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func NewConfig() *Config {
	var cfg Config
	var secret string

	flag.StringVar(&cfg.Address,
		"a", "127.0.0.1:8080",
		"Адрес, на котором располагается сервер",
	)
	flag.StringVar(&cfg.DB,
		"d", "postgresql://postgres:871023@localhost:5432/gophermart_db?sslmode=disable",
		"Адрес базы данных с которой работает сервер",
	)
	flag.StringVar(&cfg.BlackBox,
		"r", "127.0.0.1:8080/get/api/orders/",
		"Адрес черного ящика, с которой работает сервер",
	)
	flag.StringVar(&secret,
		"k", "BGCbNg8sreipgLH2",
		"Ключ для шифрования куки",
	)
	flag.Parse()

	_ = env.Parse(cfg)

	cfg.SecretCookieKey = []byte(hex.EncodeToString([]byte(secret)))

	return &cfg
}
