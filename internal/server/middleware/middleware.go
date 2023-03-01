package middleware

import (
	"github.com/gorilla/context"
	"net/http"

	"gophermart/internal/cookies"
)

type Middleware struct {
	cookie *cookies.CookieManager
}

func NewMiddleware(cookie *cookies.CookieManager) *Middleware {
	return &Middleware{
		cookie: cookie,
	}
}

func (m *Middleware) CookieChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := r.Cookies()

		login, err := m.cookie.CheckCookie(nil, cookie)
		switch err {
		case cookies.ErrNoCookie:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case cookies.ErrCipher:
			w.WriteHeader(http.StatusInternalServerError)
			return
		case cookies.ErrInvalidValue:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		context.Set(r, "login", login)

		next.ServeHTTP(w, r)
	})

}
