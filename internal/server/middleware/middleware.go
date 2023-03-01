package middleware

import (
	"github.com/gorilla/context"
	"gophermart/internal/database"
	"gophermart/internal/storage"
	"log"
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

		var err error
		var user storage.User

		cookieA := r.Cookies()

		user.Login, err = m.cookie.CheckCookie(&user, cookieA)
		switch {
		case err == database.ErrConnectToDB:
			log.Printf("Ошибка: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		case err == database.ErrRowDoesntExists:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case err == cookies.ErrNoCookie:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case err == cookies.ErrInvalidValue:
			w.WriteHeader(http.StatusBadRequest)
			return
		case err != nil:
			log.Printf("Ошибка: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
		}

		context.Set(r, "login", user.Login)

		next.ServeHTTP(w, r)
	})

}
