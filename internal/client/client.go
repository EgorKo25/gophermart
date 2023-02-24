package client

import (
	"context"
	"encoding/json"
	"errors"
	"gophermart/internal/database"
	"io"
	"net/http"
	url2 "net/url"
	"strconv"
	"time"

	"gophermart/internal/config"
	"gophermart/internal/server/handlers"
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

func (c *Client) OrdersUpdater() {

	ctx := context.Background()

	orders, _ := c.db.GetAllOrders(ctx)

	for _, order := range orders {
		switch order.Status {
		case "PROCESSING":
		case "NEW":
			c.checkOrderStatus(&order)
		default:
		}
	}

}

func (c *Client) checkOrderStatus(order *storage.Order) (int, error) {

	var body []byte

	dur := 0

	ctx := context.Background()
	url, _ := url2.JoinPath(c.cfg.BlackBox, "api", "orders", order.Number)
	timer := time.NewTimer(0)

	select {
	case <-timer.C:
		for {
			r, err := http.Get(url)
			if err != nil {
				return 0, ErrBlackBox
			}

			body, err = io.ReadAll(r.Body)
			if err != nil {
				return 0, handlers.ErrBodyRead
			}

			err = json.Unmarshal(body, order)
			if err != nil {
				return 0, handlers.ErrUnmarshal
			}

			switch r.StatusCode {
			case http.StatusOK:
				if err = c.db.SetStatus(ctx, order); err != nil {
					return 0, err
				}
				return 0, nil
			case http.StatusNoContent:
				return 0, ErrNoContent
			case http.StatusTooManyRequests:
				dur, _ = strconv.Atoi(r.Header.Get("Retry-After"))
				return dur, ErrRetryLater
			}

			return 0, ErrBlackBox
		}

	}

}