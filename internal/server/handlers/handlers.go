package handlers

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"gophermart/internal/cookies"
	"gophermart/internal/database"
	"gophermart/internal/storage"
)

type Handler struct {
	db      *database.UserDB
	coockie *cookies.CookieObj
}

func NewHandler(db *database.UserDB, cookies *cookies.CookieObj) *Handler {
	return &Handler{
		db:      db,
		coockie: cookies,
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

	h.coockie.WriteEncrypt(w, cookie)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
