package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	url2 "net/url"
	"time"

	"gophermart/internal/config"
	"gophermart/internal/database"
	"gophermart/internal/storage"
)

var (
	ErrNoContent  = errors.New("нет данных для ответа")
	ErrBlackBox   = errors.New("ошибка обращения в систему расчета")
	ErrRetryLater = errors.New("попрбобуй обратиться позже")
)

type Client struct {
	cfg *config.Config
	db  *database.UserDB
}

func NewClient(cfg *config.Config, db *database.UserDB) *Client {
	return &Client{
		cfg: cfg,
		db:  db,
	}
}

func (c *Client) OrdersUpdater() error {

	ctx := context.Background()

	orders, err := c.db.GetAllOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		switch order.Status {
		case "PROCESSING":
		case "NEW":
			c.checkOrderStatus(&order)
		case "PROCESSED":
			_ = c.db.UserBalanceUpdater(ctx, &order)
		default:
		}
	}
	return nil
}

func (c *Client) checkOrderStatus(order *storage.Order) {

	var body []byte

	ctx := context.Background()
	url, _ := url2.JoinPath(c.cfg.BlackBox, "api", "orders", order.Number)
	timer := time.NewTimer(0)

	select {
	case <-timer.C:
		for {
			r, err := http.Get(url)
			if err != nil {
				return
			}

			body, err = io.ReadAll(r.Body)
			if err != nil {
				return
			}

			err = json.Unmarshal(body, order)
			if err != nil {
				return
			}

			switch r.StatusCode {
			case http.StatusOK:
				if err = c.db.SetStatus(ctx, order); err != nil {
					return
				}
				return
			case http.StatusNoContent:
				return
			case http.StatusTooManyRequests:
				//dur, _ = strconv.Atoi(r.Header.Get("Retry-After"))
				return
			}

			return
		}

	}

}
