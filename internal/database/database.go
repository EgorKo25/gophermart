package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophermart/internal/storage"
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

	db, err := sql.Open("pgx", cfg.DB)
	if err != nil {
		log.Println("Не возожно подключиться к бд: ", err)
	}

	log.Printf("\n%s\n%s\n%s\n%s", db, cfg.DB, cfg.Address, cfg.BlackBox)
	err = createAllTablesWithContext(ctx, db)
	if err != nil {
		log.Println(err)
	}

	return &UserDB{
		db: db,
	}
}

func createAllTablesWithContext(ctx context.Context, db *sql.DB) error {

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	queries := []string{
		"CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, login VARCHAR(20), passwd VARCHAR(20));",
		"CREATE TABLE IF NOT EXISTS orders (id SERIAL PRIMARY KEY, order_title VARCHAR(20), user_token VARCHAR(20), balls INT);",
	}

	for _, query := range queries {
		r, err := db.ExecContext(childCtx, query)
		if err != nil {
			return errors.New(fmt.Sprintf("не удалось создать необходимые таблицы в базе данных. \nОшибка: %s\nОтвет базы данных: %s", err, r))
		}
	}

	return nil
}

func (d *UserDB) InsertUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if d.db == nil {
		return errors.New("отсутствует открытая база данных")
	}

	r, err := d.db.ExecContext(childCtx, "INSERT INTO users (login, passwd) VALUES($1, $2);",
		user.Login,
		user.Passwd,
	)
	if err != nil {
		return errors.New(fmt.Sprintf("не удалось отправить данные в базу данных.\n Ошибка: %s\nОтвет базы данных: %s", err, r))
	}

	return nil
}

func (d *UserDB) CheckUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT EXISTS(SELECT * FROM users WHERE login = $1 AND passwd = $2)"

	r, err := d.db.ExecContext(childCtx, query,
		user.Login,
		user.Passwd,
	)
	if err != nil {
		error := fmt.Sprintf("не удалось отправить данные в базу данных.\n Ошибка: %s\nОтвет базы данных: %s", err, r)
		return errors.New(error)
	}

	result, _ := r.RowsAffected()
	if result == 0 {
		return errors.New("пользователь не существует")
	}

	return nil

}
