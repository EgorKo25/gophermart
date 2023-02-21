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

var (
	ErrRowAlreadyExists = errors.New("запись в бд уже сущствует")
	ErrRowDoesntExists  = errors.New("записи в бд не сущетвует")
	ErrConnectToDB      = errors.New("ошибка обращения в бд")
)

type UserDB struct {
	db *sql.DB
}

func NewUserDB(cfg *config.Config) (*UserDB, error) {

	ctx := context.Background()

	db, err := sql.Open("pgx", cfg.DB)
	if err != nil {
		log.Println("Не возожно подключиться к бд: ", err)
		return nil, err
	}

	err = createAllTablesWithContext(ctx, db)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &UserDB{
		db: db,
	}, nil
}

func createAllTablesWithContext(ctx context.Context, db *sql.DB) error {

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	queries := []string{
		"CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, user_login VARCHAR(100), passwd VARCHAR(100));",
		"CREATE TABLE IF NOT EXISTS orders (id SERIAL PRIMARY KEY, user_login VARCHAR(100), order_number BIGINT);",
	}

	for _, query := range queries {
		r, err := db.ExecContext(childCtx, query)
		if err != nil {
			return errors.New(fmt.Sprintf("не удалось создать необходимые таблицы в базе данных. \nОшибка: %s\nОтвет базы данных: %s", err, r))
		}
	}

	return nil
}

func (d *UserDB) GetAllUserOrders(ctx context.Context) (result []string, err error) {

	var rows *sql.Rows

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	//query := "SELECT * FROM orders WHERE user_login = $1"

	query := "SELECT * FROM orders"

	rows, err = d.db.QueryContext(childCtx, query)
	if err != nil {
		return nil, ErrConnectToDB
	}
	//TODO:ДОДЕЛВАЙ
	result, err = rows.Columns()
	log.Println(result)
	return result, nil
}

func (d *UserDB) InsertOrderWithContext(ctx context.Context, order *storage.Order) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "INSERT INTO orders (user_login, order_number) VALUES($1, $2);"

	_, err := d.db.ExecContext(childCtx, query, order.User, order.Number)
	if err != nil {
		return ErrConnectToDB
	}

	return nil

}

func (d *UserDB) CheckOrderWithContext(ctx context.Context, order *storage.Order) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT EXISTS(SELECT * FROM orders WHERE user_login = $1 AND order_number = $2)"

	r, err := d.db.ExecContext(childCtx, query,
		order.User,
		order.Number,
	)

	if err != nil {
		return ErrConnectToDB
	}

	result, _ := r.RowsAffected()
	if result == 0 {
		return ErrRowDoesntExists
	}

	return nil

}

/*
	func (d *UserDB) CheckOrderWithContext(ctx context.Context, ) error {
		childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		query := "SELECT EXISTS(SELECT * FROM orders WHERE user_login = $1 AND order_number = $2);"

		r, err := d.db.ExecContext(childCtx, query, order.User, order.Number)
		if err != nil {
			log.Println(r, err)
			return ErrConnectToDB
		}

		result, _ := r.RowsAffected()
		if result != 0 {
			return ErrRowAlreadyExists
		}

		return nil

}
*/
func (d *UserDB) InsertUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if d.db == nil {
		return errors.New("отсутствует открытая база данных")
	}

	r, err := d.db.ExecContext(childCtx, "INSERT INTO users (user_login, passwd) VALUES($1, $2);", user.Login, user.Passwd)
	if err != nil {
		return errors.New(fmt.Sprintf("не удалось отправить данные в базу данных.\n Ошибка: %s\nОтвет базы данных: %s", err, r))
	}

	return nil
}

func (d *UserDB) CheckUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT EXISTS(SELECT * FROM users WHERE user_login = $1 AND passwd = $2)"

	r, err := d.db.ExecContext(childCtx, query,
		user.Login,
		user.Passwd,
	)
	if err != nil {
		return ErrConnectToDB
	}

	result, _ := r.RowsAffected()
	if result == 0 {
		return ErrRowDoesntExists
	}

	return nil

}
