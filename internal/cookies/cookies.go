package cookies

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"gophermart/internal/database"
	"gophermart/internal/storage"
)

type CookieManager struct {
	Key []byte
	db  *database.UserDB
}

func NewCookieManager(key []byte, db *database.UserDB) *CookieManager {
	return &CookieManager{
		Key: key,
		db:  db,
	}
}

var (
	ErrValueTooLong = errors.New("слишком длинное значение куки")
	ErrInvalidValue = errors.New("неверное значение куки")
	ErrCipher       = errors.New("ошибка шифрования куки")
	ErrNoCookie     = errors.New("нет куки")
	ErrGobZip       = errors.New("ошибка упаковки в gob")
)

func (c *CookieManager) Write(cookie http.Cookie) (*http.Cookie, error) {

	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

	if len(cookie.String()) > 4096 {
		return nil, ErrValueTooLong
	}

	return &cookie, nil
}

func (c *CookieManager) Read(cookie *http.Cookie) (string, error) {

	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidValue
	}

	return string(value), nil
}

func (c *CookieManager) signMyDate(name, value string) string {

	hm := hmac.New(sha256.New, c.Key)
	hm.Write([]byte(name))
	hm.Write([]byte(value))
	signature := hm.Sum(nil)

	value = string(signature) + value

	return value
}

func (c *CookieManager) WriteEncrypt(cookie http.Cookie) (*http.Cookie, error) {

	cookie.Value = c.signMyDate(cookie.Name, cookie.Value)

	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, ErrCipher
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrCipher
	}

	nonce := make([]byte, aesGCM.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, ErrCipher
	}

	plaintext := fmt.Sprintf("%s:%s", cookie.Name, cookie.Value)

	encryptedValue := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	cookie.Value = string(encryptedValue)

	return c.Write(cookie)
}

func (c *CookieManager) ReadEncrypt(cookie *http.Cookie, name string, secretKey []byte) (string, error) {
	encryptedValue, err := c.Read(cookie)
	if err != nil {
		return "", ErrInvalidValue
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", ErrCipher
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrCipher
	}

	nonceSize := aesGCM.NonceSize()

	if len(encryptedValue) < nonceSize {
		return "", ErrInvalidValue
	}

	nonce := encryptedValue[:nonceSize]
	ciphertext := encryptedValue[nonceSize:]

	plaintext, err := aesGCM.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return "", ErrInvalidValue
	}

	expectedName, value, ok := strings.Cut(string(plaintext), ":")
	if !ok {
		return "", ErrInvalidValue
	}

	if len(value) < sha256.Size {
		return "", ErrInvalidValue
	}

	signature := value[:sha256.Size]
	value = value[sha256.Size:]

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(cookie.Name))
	mac.Write([]byte(value))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal([]byte(signature), expectedSignature) {
		return "", ErrInvalidValue
	}

	if expectedName != name {
		return "", ErrInvalidValue
	}

	return value, nil
}
func (c *CookieManager) GetCookie(user *storage.User) (final *http.Cookie, err error) {
	var buffer bytes.Buffer

	cookie := http.Cookie{
		Name:  fmt.Sprintf("CookieUser%s", user.Login),
		Value: user.Login,

		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}

	err = gob.NewEncoder(&buffer).Encode(cookie)
	if err != nil {
		log.Printf("Что-то не так:\n%e", ErrGobZip)
		return nil, ErrGobZip
	}

	final, err = c.WriteEncrypt(cookie)
	if err != nil {
		return nil, err
	}

	return final, nil

}

func (c *CookieManager) CheckCookie(user *storage.User, cookieAll []*http.Cookie) (string, error) {

	ctx := context.Background()

	for _, cookie := range cookieAll {
		if cookie != nil {
			value, err := c.ReadEncrypt(cookie, cookie.Name, c.Key)
			switch err {
			case ErrCipher:
				err = ErrCipher
			case ErrInvalidValue:
				err = ErrInvalidValue
			default:
				return value, nil
			}
		}
	}

	if user != nil {
		err := c.db.CheckUserWithContext(ctx, user)
		switch err {
		case database.ErrConnectToDB:
			return "", database.ErrConnectToDB

		case database.ErrRowDoesntExists:
			return "", database.ErrRowDoesntExists
		case nil:
			return "", nil
		}

	}

	return "", ErrNoCookie

}
