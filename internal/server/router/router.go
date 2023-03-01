package router

import (
	"gophermart/internal/server/handlers"
	"gophermart/internal/server/middleware"

	"github.com/go-chi/chi/v5"
	mdw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *handlers.Handler, middle *middleware.Middleware) chi.Router {
	r := chi.NewRouter()

	r.Use(mdw.Logger)

	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", handler.Register)
		r.Post("/api/user/login", handler.Login)
	})
	r.Group(func(r chi.Router) {
		r.Use(middle.CookieChecker)

		r.Get("/api/user/orders", handler.AllOrder)
		r.Get("/api/user/withdrawals", handler.AllWithdrawals)
		r.Get("/api/user/balance", handler.Balance)

		r.Post("/api/user/orders", handler.Orders)
		r.Post("/api/user/balance/withdraw", handler.Withdraw)
	})

	return r
}
