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
	db            *database.UserDB
	coockieFormat *cookies.CoockieFormat
}

func NewHandler(db *database.UserDB, format *cookies.CoockieFormat) *Handler {
	return &Handler{
		db:            db,
		coockieFormat: format,
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
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer func() {
		err = r.Body.Close()
	}()

	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Printf("Ошибка перевода из формата json: \n%e", err)
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
