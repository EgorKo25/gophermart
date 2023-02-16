package cookies

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type CookieObj struct {
	key []byte
}

func NewCookieObj(key []byte) *CookieObj {
	return &CookieObj{
		key: key,
	}
}

var (
	ErrValueTooLong = errors.New("слишком длинное значение куки")
	ErrInvalidValue = errors.New("неверное значение куки")
)

func (c *CookieObj) Write(w http.ResponseWriter, cookie http.Cookie) error {

	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

	if len(cookie.String()) > 4096 {
		return ErrValueTooLong
	}

	http.SetCookie(w, &cookie)

	return nil
}

func (c *CookieObj) Read(r *http.Request, name string) (string, error) {

	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	// Decode the base64-encoded cookie value. If the cookie didn't contain a
	// valid base64-encoded value, this operation will fail and we return an
	// ErrInvalidValue error.
	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidValue
	}

	return string(value), nil
}

func (c *CookieObj) signMyDate(name, value string) string {

	hm := hmac.New(sha256.New, c.key)
	hm.Write([]byte(name))
	hm.Write([]byte(value))
	signature := hm.Sum(nil)

	value = string(signature) + value

	return value
}

func (c *CookieObj) WriteEncrypt(w http.ResponseWriter, cookie http.Cookie) error {

	cookie.Value = c.signMyDate(cookie.Name, cookie.Value)

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return err
	}

	plaintext := fmt.Sprintf("%s:%s", cookie.Name, cookie.Value)

	encryptedValue := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	cookie.Value = string(encryptedValue)

	return c.Write(w, cookie)
}
