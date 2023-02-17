package handlers

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal/cookies"
	"gophermart/internal/database"
	"gophermart/internal/storage"
	"io"
	"log"
	"net/http"
	"strconv"
)

var (
	ErrUnmarshal = errors.New("ошибка десериализации/сериализации")
	ErrNoCookie  = errors.New("нет куки")
)

type Handler struct {
	db      *database.UserDB
	cookies *cookies.CookieObj
}

func NewHandler(db *database.UserDB, cookies *cookies.CookieObj) *Handler {
	return &Handler{
		db:      db,
		cookies: cookies,
	}
}

func (h *Handler) MainPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdrawals(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Balance(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var user storage.User
	ctx := context.Background()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("Не удалось закрыть тело запроса: \n%e", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Printf("%e", ErrUnmarshal)
		w.WriteHeader(http.StatusBadRequest)
	}

	err = h.db.InsertUserWithContext(ctx, &user)
	if err != nil {
		log.Printf("Ошибка при обращении в бд: \n%e", err)
		w.WriteHeader(http.StatusConflict)
	}

	h.getCookie(w, &user)

}
func (h *Handler) getCookie(w http.ResponseWriter, user *storage.User) {
	var buffer bytes.Buffer
	var err error

	cookie := http.Cookie{
		Name:  fmt.Sprintf("CookieUser%s", user.Login),
		Value: user.Login,

		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}

	err = gob.NewEncoder(&buffer).Encode(cookie)
	if err != nil {
		log.Printf("Что-то не так:\n%e", errors.New("ошибка упаковки в gob"))
	}

	err = h.cookies.WriteEncrypt(w, cookie)
	if err != nil {
		log.Printf("%e", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	var user storage.User
	ctx := context.Background()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("Не удалось закрыть тело запроса: \n%e", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if len(body) > 0 {
		err = json.Unmarshal(body, &user)
		if err != nil {
			log.Printf("Ошибка перевода из формата json: \n%e", err)
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	_, err = h.checkCookie(ctx, r, &user)

	switch {
	case err == ErrNoCookie:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err == database.ErrRowDoesntExists:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err == database.ErrConnectToDB:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	case err == cookies.ErrInvalidValue:
		w.WriteHeader(http.StatusBadRequest)
	case err != nil:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	default:
		h.getCookie(w, &user)
		w.WriteHeader(http.StatusOK)
	}

}

func (h *Handler) checkCookie(ctx context.Context, r *http.Request, user *storage.User) (string, error) {
	cookiesAll := r.Cookies()

	if user != nil {
		err := h.db.CheckUserWithContext(ctx, user)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	for _, cookie := range cookiesAll {
		if cookie != nil {
			value, err := h.cookies.ReadEncrypt(r, cookie.Name, h.cookies.Key)
			if err != nil {
				return "", err
			}
			return value, nil
		}
	}

	return "", ErrNoCookie

}

func (h *Handler) Order(w http.ResponseWriter, r *http.Request) {

	var order storage.Order
	ctx := context.Background()

	body, err := io.ReadAll(r.Body)

	order.Number, err = strconv.Atoi(string(body))

	order.User, err = h.checkCookie(ctx, r, nil)

	switch {
	case err == ErrNoCookie:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err == database.ErrRowDoesntExists:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err == database.ErrConnectToDB:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	case err == cookies.ErrInvalidValue:
		w.WriteHeader(http.StatusBadRequest)
	case err != nil:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.db.CheckOrderWithContext(ctx, &order)
	switch err {
	case database.ErrConnectToDB:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	case database.ErrRowAlreadyExists:
		w.WriteHeader(http.StatusOK)
	default:
		err = h.db.InsertOrderWithContext(ctx, &order)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
