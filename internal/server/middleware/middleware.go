package middleware

import (
	"encoding/json"
	"github.com/gorilla/context"
	"gophermart/internal/database"
	"gophermart/internal/storage"
	"io"
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

		var user storage.User

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal(body, &user)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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
