package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"gophermart/internal/config"
	"gophermart/internal/database"
	"gophermart/internal/storage"
)

var (
	ErrBlackBox = errors.New("ошибка обращения в систему расчета")
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

func (c *Client) Run() (err error) {
	for {
		err = c.OrdersUpdater()
		if err != nil {
			log.Printf("%s", err)
			return err
		}
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
		default:
		}
	}
	return nil
}

func (c *Client) checkOrderStatus(order *storage.Order) {

	var body []byte

	ctx := context.Background()
	addr, _ := url.JoinPath(c.cfg.BlackBox, "api", "orders", order.Number)
	timer := time.NewTimer(0)

	select {
	case <-timer.C:
		for {
			r, err := http.Get(addr)
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
				return
			}

			return
		}

	}

}
