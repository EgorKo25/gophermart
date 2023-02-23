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
		"CREATE TABLE IF NOT EXISTS orders (id SERIAL PRIMARY KEY, user_login VARCHAR(100), order_number BIGINT, status VARCHAR(10), accrual FLOAT, uploaded_at VARCHAR(50));",
	}

	for _, query := range queries {
		r, err := db.ExecContext(childCtx, query)
		if err != nil {
			return errors.New(fmt.Sprintf("не удалось создать необходимые таблицы в базе данных. \nОшибка: %s\nОтвет базы данных: %s", err, r))
		}
	}

	return nil
}
func (d *UserDB) SetStatus(ctx context.Context, order *storage.Order) error {

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "UPDATE orders SET status = $1, accrual = $2 WHERE order_number = $3"

	_, err := d.db.ExecContext(childCtx, query,
		order.Status,
		order.Accrual,
		order.Number,
	)
	if err != nil {
		return ErrConnectToDB
	}

	return nil

}
func (d *UserDB) GetAllUserOrders(ctx context.Context, user *storage.User) (orders []storage.Order, err error) {

	var ord storage.Order

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT * FROM orders WHERE user_login = $1"

	rows, err := d.db.QueryContext(childCtx, query,
		user.Login,
	)
	if err != nil {
		return orders, ErrConnectToDB
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&ord.ID, &ord.User, &ord.Number, &ord.Status,
			&ord.Accrual, &ord.Uploaded_at,
		); err != nil {
			return orders, err
		}

		orders = append(orders, ord)
	}
	if err = rows.Err(); err != nil {
		return orders, err
	}
	return orders, nil
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

	var result bool

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT EXISTS(SELECT * FROM orders WHERE user_login = $1 AND order_number = $2)"

	r, err := d.db.QueryContext(childCtx, query,
		order.User,
		order.Number,
	)

	if err != nil {
		return ErrConnectToDB
	}

	r.Next()
	r.Scan(&result)

	if result {
		return ErrRowAlreadyExists
	}

	return nil

}

func (d *UserDB) InsertUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if d.db == nil {
		return ErrConnectToDB
	}

	_, err := d.db.ExecContext(childCtx, "INSERT INTO users (user_login, passwd) VALUES($1, $2);", user.Login, user.Passwd)
	if err != nil {
		return ErrConnectToDB
	}

	return nil
}

func (d *UserDB) CheckUserWithContext(ctx context.Context, user *storage.User) error {

	var result bool

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT EXISTS(SELECT * FROM users WHERE user_login = $1 AND passwd = $2)"

	r, err := d.db.QueryContext(childCtx, query,
		user.Login,
		user.Passwd,
	)
	if err != nil {
		return ErrConnectToDB
	}

	r.Next()
	r.Scan(&result)

	if result {
		return nil
	}

	return ErrRowDoesntExists

}
