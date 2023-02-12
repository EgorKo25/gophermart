package database

import (
	"context"
	"database/sql"
	"log"
	"time"

	"gophermart/internal/config"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type UserDB struct {
	db *sql.DB
}

func NewUserDB(cfg *config.Config) *UserDB {

	ctx := context.Background()

	databaseAddr := cfg.Address + ":" + cfg.Port

	db, err := sql.Open("pgx", databaseAddr)
	if err != nil {
		log.Println("Не возожно подключиться к бд: ", err)
	}

	err = createAllTablesWithContext(ctx, db)
	if err != nil {
		log.Println("Не удалось создать необходимые таблицы: ", err)
	}

	return &UserDB{
		db: db,
	}
}

func createAllTablesWithContext(ctx context.Context, db *sql.DB) error {

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	queries := []string{
		"CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, firstname VARCHAR(20), lastname VARCHAR(20), login VARCHAR(20), passwd VARCHAR(20), token VARCHAR(50), balance BIGINT);",
		"CREATE TABLE IF NOT EXISTS orders (id SERIAL PRIMARY KEY, order_title VARCHAR(20), user_token VARCHAR(20), balls INT);",
	}

	for _, query := range queries {
		r, err := db.ExecContext(childCtx, query)
		if err != nil {
			log.Println("Не удалось создать необходимые таблицы в базе данных. \nОшибка: ", err, "\nОтвет базы данных: ", r)
			return err
		}
	}

	return nil
}
