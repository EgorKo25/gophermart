package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/theplant/luhn"
	"gophermart/internal/config"
	"gophermart/internal/cookies"
	"gophermart/internal/database"
	"gophermart/internal/storage"

	gctx "github.com/gorilla/context"
)

var (
	ErrUnmarshal = errors.New("ошибка десериализации/сериализации")
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

func (h *Handler) AllWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var resp []byte
	var err error
	var user storage.User
	var withdrawalsList []storage.Withdraw

	user.Login = gctx.Get(r, "login").(string)
	log.Println("middleware has worked.... login is ", user.Login)

	withdrawalsList, err = h.db.GetAllWithdraw(ctx, &user)
	switch err {
	case database.ErrConnectToDB:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case nil:
		if len(withdrawalsList) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(resp)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		log.Println(withdrawalsList)
		resp, err = json.Marshal(withdrawalsList)
		if err != nil {
			log.Printf("%e: %e", ErrUnmarshal, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {

	var user storage.User
	var err error
	var body []byte

	cookieA := r.Cookies()
	user.Login, err = h.cookies.CheckCookie(&user, cookieA)

	switch {
	case err == database.ErrConnectToDB:
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	default:
	}

	user.Balance, user.Withdraw, err = h.db.GetBall(user.Login)
	if err != nil {
		log.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println(user)
	body, err = json.Marshal(user)
	if err != nil {
		log.Printf("%e: %e", ErrUnmarshal, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
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

	var body []byte
	var order storage.Order
	var err error

	ctx := context.Background()

	order.User = gctx.Get(r, "login").(string)
	order.Status = "NEW"
	order.UploadedAt = time.Now().Format(time.RFC3339)

	body, err = io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	order.Number = string(body)

	err = h.luhnCheck(order.Number)
	if err != nil {
		log.Printf("%s", err)
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
	case database.ErrRowWasCreatedAnyUser:
		w.WriteHeader(http.StatusConflict)
		return
	default:
	}

	err = h.db.InsertOrderWithContext(ctx, &order)
	if err != nil {
		log.Printf("%s: %s", database.ErrConnectToDB, err)
		w.WriteHeader(http.StatusInternalServerError)
		return

	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) luhnCheck(number string) error {
	tmp, _ := strconv.Atoi(number)
	if isValid := luhn.Valid(tmp); !isValid {
		return database.ErrNumberFormat
	}
	return nil
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {

	var err error
	var user storage.User
	var withdraw storage.Withdraw
	var body []byte

	ctx := context.Background()

	withdraw.User = gctx.Get(r, "login").(string)

	body, err = io.ReadAll(r.Body)
	if err != nil {
		log.Printf("%s: %s", ErrBodyRead, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		err = r.Body.Close()
		if err != nil {
			log.Printf("%s", ErrBodyRead)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	err = json.Unmarshal(body, &withdraw)
	if err != nil {
		log.Printf("%s: %s", ErrUnmarshal, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.luhnCheck(withdraw.NumberOrder)
	if err != nil {
		log.Printf("%s", err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	withdraw.ProcessedAt = time.Now().Format(time.RFC3339)

	err = h.db.Withdraw(ctx, &user, &withdraw)
	switch err {
	case database.ErrConnectToDB:
		log.Printf("%s: %s", database.ErrConnectToDB, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	case database.ErrNotEnoughMoney:
		w.WriteHeader(http.StatusPaymentRequired)
		return
	default:
	}

	w.WriteHeader(http.StatusOK)

}

func (h *Handler) AllOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var resp []byte
	var err error
	var orderList []storage.Order

	login := gctx.Get(r, "login").(string)

	orderList, err = h.db.GetAllUserOrders(ctx, login)
	switch err {
	case database.ErrConnectToDB:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case nil:
		if len(orderList) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(resp)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp, err = json.Marshal(orderList)
		if err != nil {
			log.Printf("%e: %e", ErrUnmarshal, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
		w.WriteHeader(http.StatusOK)
		return
	}

}
