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
)

type Handler struct {
	db       *database.UserDB
	coockies *cookies.CookieObj
}

func NewHandler(db *database.UserDB, cookies *cookies.CookieObj) *Handler {
	return &Handler{
		db:       db,
		coockies: cookies,
	}
}

func (h *Handler) MainPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdrawals(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var buffer bytes.Buffer
	var user storage.User
	var ctx context.Context

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
		log.Printf("Ошибка перевода из формата json: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
	}

	err = h.db.InsertUserWithContext(ctx, &user)
	if err != nil {
		log.Printf("Ошибка при обращении в бд: \n%e", err)
		w.WriteHeader(http.StatusConflict)
	}

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

	h.coockies.WriteEncrypt(w, cookie)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	var user storage.User
	var ctx context.Context

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
		log.Printf("Ошибка перевода из формата json: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
	}

	cookiesAll := r.Cookies()

	_, err = h.coockies.ReadEncrypt(r, cookiesAll[0].Name, h.coockies.Key)
	if err != nil {
		switch err {
		case cookies.ErrInvalidValue:
			log.Printf("Ошибка: %e", errors.New("куки изменены"))
			w.WriteHeader(http.StatusBadRequest)

		case http.ErrNoCookie:
			err = h.db.CheckUserWithContext(ctx, &user)
			if err != nil {
				log.Printf("Ошибка: %e", err)
				w.WriteHeader(http.StatusUnauthorized)
			}
		}

	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
