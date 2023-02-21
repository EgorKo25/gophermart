package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal/config"
	"gophermart/internal/cookies"
	"gophermart/internal/database"
	"gophermart/internal/storage"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/theplant/luhn"
)

var (
	ErrUnmarshal = errors.New("ошибка десериализации/сериализации")
	ErrBlackBox  = errors.New("ошибка обращения в систему расчета")
	ErrBodyRead  = errors.New("ошибка чтения ответа")
	ErrBodyClose = errors.New("неудалось закрыть тело запроса")
)

type Handler struct {
	db      *database.UserDB
	cookies *cookies.CookieManager
	cfg     *config.Config
}

func NewHandler(db *database.UserDB, cookies *cookies.CookieManager) *Handler {
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
	var cookie *http.Cookie
	ctx := context.Background()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("Не удалось закрыть тело запроса: \n%e", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}()

	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Printf("%e", ErrUnmarshal)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.db.InsertUserWithContext(ctx, &user)
	if err != nil {
		log.Printf("Ошибка при обращении в бд: \n%e", err)
		w.WriteHeader(http.StatusConflict)
		return
	}

	cookie, err = h.cookies.GetCookie(&user)
	switch err {
	case cookies.ErrValueTooLong:
		w.WriteHeader(http.StatusBadRequest)
		return
	case cookies.ErrCipher:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case nil:
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("неизвестная ошибка %s", err)
		return
	}

}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	var user storage.User
	var cookie *http.Cookie

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: \n%e", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("Не удалось закрыть тело запроса: \n%e", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}()

	if len(body) > 0 {
		err = json.Unmarshal(body, &user)
		if err != nil {
			log.Printf("Ошибка перевода из формата json: \n%e", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	cookieA := r.Cookies()
	_, err = h.cookies.CheckCookie(&user, cookieA)

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

	cookie, err = h.cookies.GetCookie(&user)
	switch err {
	case cookies.ErrValueTooLong:
		w.WriteHeader(http.StatusBadRequest)
		return
	case cookies.ErrCipher:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case nil:
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("неизвестная ошибка %s", err)
		return
	}

}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {

	var order storage.Order
	var body []byte
	var err error

	ctx := context.Background()

	body, err = io.ReadAll(r.Body)
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("%s", ErrBodyRead)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	order.Number, err = strconv.Atoi(fmt.Sprintf("%s", body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isValid := luhn.Valid(order.Number); isValid == false {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	log.Println(order)
	order.Status = "NEW"
	order.Accrual = 0.0

	err = h.db.CheckOrderWithContext(ctx, &order)
	switch err {
	case database.ErrConnectToDB:
		log.Printf("Ошибка: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	case database.ErrRowAlreadyExists:
		w.WriteHeader(http.StatusOK)
	case nil:
		err = h.db.InsertOrderWithContext(ctx, &order)
		w.WriteHeader(http.StatusAccepted)

		go h.checkOrderStatus(&order)

		return
	}
}

func (h *Handler) checkOrderStatus(order *storage.Order) (err error) {

	var r *http.Response
	var body []byte
	var dur int

	ctx := context.Background()
	url := h.cfg.BlackBox + strconv.Itoa(order.Number)

	timer := time.NewTimer(time.Duration(dur))
	for {
		select {
		case <-timer.C:
			r, err = http.Get(url)

			if err != nil {
				log.Printf("%s", ErrBlackBox)
				return ErrBlackBox
			}

			body, err = io.ReadAll(r.Body)
			if err != nil {
				log.Printf("%s", ErrBodyRead)
				return ErrBodyRead
			}

			err = json.Unmarshal(body, &order)
			if err != nil {
				log.Printf("%s", ErrUnmarshal)
				return ErrUnmarshal
			}

			switch r.StatusCode {
			case http.StatusOK:
				h.db.SetStatus(ctx, order)
				return nil
			case http.StatusTooManyRequests:
				dur, err = strconv.Atoi(r.Header.Get("Retry-After"))
				if err != nil {
					log.Printf("%s", err)
					return err
				}

				timer = time.NewTimer(time.Duration(dur))
			default:
				h.db.SetStatus(ctx, order)
				return
			}
		}
	}
}

func (h *Handler) Withdraw(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) AllOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var err error
	var user storage.User

	cookie := r.Cookies()

	user.Login, err = h.cookies.CheckCookie(nil, cookie)
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

	ordersList, err := h.db.GetAllUserOrders(ctx, &user)
	log.Println(*ordersList)
}
