package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	url2 "net/url"
	"strconv"
	"time"

	"gophermart/internal/config"
	"gophermart/internal/cookies"
	"gophermart/internal/database"
	"gophermart/internal/storage"

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

func NewHandler(db *database.UserDB, cookies *cookies.CookieManager, cfg *config.Config) *Handler {
	return &Handler{
		db:      db,
		cookies: cookies,
		cfg:     cfg,
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
		log.Printf("%s", ErrBodyRead)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("%s", ErrBodyClose)
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
		log.Printf("%s", database.ErrConnectToDB)
		w.WriteHeader(http.StatusConflict)
		return
	}

	cookie, err = h.cookies.GetCookie(&user)
	log.Println(cookie)
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

	if len(body) > 0 {
		err = json.Unmarshal(body, &user)
		if err != nil {
			log.Printf("Ошибка перевода из формата json: \n%e", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = h.db.CheckUserWithContext(ctx, &user)
		switch err {
		case database.ErrConnectToDB:
			w.WriteHeader(http.StatusInternalServerError)
			return
		case database.ErrRowDoesntExists:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case nil:
			cookie, err = h.cookies.GetCookie(&user)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, cookie)
			w.WriteHeader(http.StatusOK)
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

	var user storage.User
	var order storage.Order
	var body []byte
	var err error

	ctx := context.Background()

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

	body, err = io.ReadAll(r.Body)
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("%s", ErrBodyRead)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	order.Number = fmt.Sprintf("%s", body)

	tmp, _ := strconv.Atoi(order.Number)
	if isValid := luhn.Valid(tmp); isValid == false {
		w.WriteHeader(http.StatusUnprocessableEntity)
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
		return
	case nil:
		err = h.db.InsertOrderWithContext(ctx, &order)
		if err != nil {
			log.Printf("%s", err)
		}

		go func() {
			err = h.checkOrderStatus(&order)
			if err != nil {
				log.Printf("%s", err)
			}
		}()

		w.WriteHeader(http.StatusAccepted)
		return
	}
}

func (h *Handler) checkOrderStatus(order *storage.Order) error {

	var body []byte

	dur := 0

	ctx := context.Background()
	url, _ := url2.JoinPath(h.cfg.BlackBox, "api", "orders", order.Number)
	timer := time.NewTimer(time.Duration(dur))

	for {
		select {
		case <-timer.C:

			r, err := http.Get(url)
			if err != nil {
				return ErrBlackBox
			}

			body, err = io.ReadAll(r.Body)
			if err != nil {
				log.Printf("%s", ErrBodyRead)
				return ErrBodyRead
			}

			err = json.Unmarshal(body, order)
			if err != nil {
				return ErrUnmarshal
			}

			log.Println(order)

			switch r.StatusCode {
			case http.StatusNoContent:
				return nil
			case http.StatusOK:
				if err = h.db.SetStatus(ctx, order); err != nil {
					return err
				}
				return nil
			case http.StatusTooManyRequests:
				dur, err = strconv.Atoi(r.Header.Get("Retry-After"))
				if err != nil {
					log.Printf("%s", err)
					return err
				}

				timer = time.NewTimer(time.Duration(dur))
			}
		}
	}
}

func (h *Handler) Withdraw(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) AllOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var resp []byte
	var err error
	var user storage.User
	var orderList []storage.Order

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

	orderList, err = h.db.GetAllUserOrders(ctx, &user)
	switch err {
	case database.ErrConnectToDB:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case nil:
		log.Println(orderList)

		if len(orderList) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp, err = json.Marshal(orderList)
		if err != nil {
			log.Printf("%s: %s", ErrUnmarshal, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Println(err)
		log.Println(orderList)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}
