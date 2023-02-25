package router

import (
	"gophermart/internal/server/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *handlers.Handler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Route("/", func(r chi.Router) {
		r.Get("/", handler.MainPage)
		r.Get("/api/user/orders", handler.AllOrder)
		r.Get("/api/user/withdrawals", handler.Withdrawals)
		r.Get("/api/user/balance", handler.Balance)

		r.Post("/api/user/register", handler.Register)
		r.Post("/api/user/login", handler.Login)
		r.Post("/api/user/orders", handler.Orders)
		r.Post("/api/user/balance/withdraw", handler.Withdraw)

	})

	return r
}
