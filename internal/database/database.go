package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"log"
	"time"

	"gophermart/internal/config"
	"gophermart/internal/storage"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrNotEnoughMoney = errors.New("не достаточно средств на счету")
	ErrNumberFormat   = errors.New("алгоритм луна выявил неправильный формат заказа")

	ErrRowAlreadyExists     = errors.New("запись в бд уже сущствует")
	ErrRowDoesntExists      = errors.New("записи в бд не сущетвует")
	ErrConnectToDB          = errors.New("ошибка обращения в бд")
	ErrRowWasCreatedAnyUser = errors.New("дургой пользователь уже добавил номер этого заказа")
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
		"CREATE TABLE IF NOT EXISTS withdrawals (id SERIAL PRIMARY KEY, number BIGINT, sum FLOAT, processed_at VARCHAR(50), user_login VARCHAR(20))",
		"CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, user_login VARCHAR(100), passwd VARCHAR(100), balance FLOAT, withdrow FLOAT);",
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

func (d *UserDB) GetBall(user string) (bal, with float64, err error) {

	query := "SELECT balance, withdrow FROM users WHERE user_login = $1;"

	r, err := d.db.Query(query, user)
	if err != nil {
		return 0, 0, ErrConnectToDB
	}

	r.Next()
	_ = r.Scan(&bal, &with)

	log.Println(bal, with)
	return bal, with, nil
}

func (d *UserDB) Withdraw(ctx context.Context, user *storage.User, withdraw *storage.Withdraw) error {
	childCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	bal, with, err := d.GetBall(withdraw.User)
	if err != nil {
		return err
	}

	balance := bal - withdraw.Sum
	log.Println(balance, withdraw, user)
	/*
		balance := bal - withdraw.Sum
		if balance <= 0 {
			return ErrNotEnoughMoney
		}
	*/
	query := "UPDATE users SET withdrow = $1, balance = $2 WHERE user_login = $3"

	_, err = d.db.ExecContext(childCtx, query,
		with+withdraw.Sum,
		balance,
		withdraw.User,
	)
	if err != nil {
		return err
	}

	query = "INSERT INTO withdrawals (number, sum, processed_at, user_login) VALUES($1, $2, $3, $4);"

	_, err = d.db.ExecContext(childCtx, query,
		withdraw.NumberOrder,
		withdraw.Sum,
		withdraw.ProcessedAt,
		withdraw.User,
	)
	if err != nil {
		return err
	}

	return nil

}

func (d *UserDB) sortDate(login string) error {
	query := `
			SELECT user_login, processed_at 
			FROM withdrawals 
			WHERE user_login = $1
			ORDER BY processed_at
	`

	_, err := d.db.Exec(query, login)
	if err != nil {
		return err
	}

	return nil
}
func (d *UserDB) GetAllWithdraw(ctx context.Context, user *storage.User) (withdrawals []storage.Withdraw, err error) {

	var wtd storage.Withdraw
	var rows *sql.Rows

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	d.sortDate(user.Login)

	query := "SELECT * FROM withdrawals WHERE user_login = $1"

	rows, err = d.db.QueryContext(childCtx, query, user.Login)
	if err != nil {
		return withdrawals, err
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&wtd.ID, &wtd.NumberOrder, &wtd.ProcessedAt); err != nil {
			return withdrawals, err
		}

		withdrawals = append(withdrawals, wtd)
	}
	if err = rows.Err(); err != nil {
		return withdrawals, err
	}
	return withdrawals, nil
}

func (d *UserDB) UserBalanceUpdater(ctx context.Context, order *storage.Order) error {

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	bal, _, err := d.GetBall(order.User)
	if err != nil {
		return err
	}

	query := "UPDATE users SET balance = $1 WHERE user_login = $2"

	_, err = d.db.ExecContext(childCtx, query,
		order.Accrual+bal,
		order.User,
	)
	if err != nil {
		return err
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
		return err
	}

	if order.Status == "PROCESSED" {
		bal, _, _ := d.GetBall(order.User)
		query = "UPDATE users SET balance = $1 WHERE user_login = $2"

		_, err = d.db.ExecContext(childCtx, query,
			order.Accrual+bal,
			order.User,
		)
		if err != nil {
			return err
		}
	}

	return nil

}

func (d *UserDB) GetAllOrders(ctx context.Context) (orders []storage.Order, err error) {

	var ord storage.Order
	var rows *sql.Rows

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT * FROM orders"

	rows, err = d.db.QueryContext(childCtx, query)
	if err != nil {
		return orders, ErrConnectToDB
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&ord.ID, &ord.User, &ord.Number, &ord.Status,
			&ord.Accrual, &ord.UploadedAt,
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

func (d *UserDB) GetAllUserOrders(ctx context.Context, user *storage.User) (orders []storage.Order, err error) {

	var ord storage.Order

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := "SELECT * FROM orders WHERE user_login = $1"

	rows, err := d.db.QueryContext(childCtx, query,
		user.Login,
	)
	if err != nil {
		return orders, err
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&ord.ID, &ord.User, &ord.Number, &ord.Status,
			&ord.Accrual, &ord.UploadedAt,
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

	query := "INSERT INTO orders (user_login, order_number, status, accrual, uploaded_at) VALUES($1, $2, $3, $4, $5);"

	_, err := d.db.ExecContext(childCtx, query,
		order.User,
		order.Number,
		order.Status,
		order.Accrual,
		order.UploadedAt,
	)
	if err != nil {
		return err
	}

	return nil

}

func (d *UserDB) CheckOrderWithContext(ctx context.Context, order *storage.Order) error {

	var result bool

	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	queries := []string{"SELECT EXISTS(SELECT * FROM orders WHERE user_login = $1 AND order_number = $2)",
		"SELECT EXISTS(SELECT * FROM orders WHERE order_number = $1)",
	}
	r, err := d.db.QueryContext(childCtx, queries[0],
		order.User,
		order.Number,
	)

	if err != nil {
		return ErrConnectToDB
	}

	r.Next()
	_ = r.Scan(&result)

	if result {
		return ErrRowAlreadyExists
	}
	r, err = d.db.QueryContext(childCtx, queries[1],
		order.Number,
	)

	if err != nil {
		return ErrConnectToDB
	}

	r.Next()
	_ = r.Scan(&result)

	if result {
		return ErrRowWasCreatedAnyUser
	}

	return nil

}

func (d *UserDB) InsertUserWithContext(ctx context.Context, user *storage.User) error {
	childCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if d.db == nil {
		return ErrConnectToDB
	}

	_, err := d.db.ExecContext(childCtx, "INSERT INTO users (user_login, passwd, balance, withdrow) VALUES($1, $2, $3, $4);",
		user.Login,
		user.Passwd,
		user.Balance,
		user.Withdraw,
	)
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
	_ = r.Scan(&result)

	if result {
		return nil
	}

	return ErrRowDoesntExists

}
