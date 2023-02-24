package cookies

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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

func (c *CookieManager) WriteEncrypt(cookie http.Cookie) (*http.Cookie, error) {

	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	// Wrap the cipher block in Galois Counter Mode.
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create a unique nonce containing 12 random bytes.
	nonce := make([]byte, aesGCM.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	// Prepare the plaintext input for encryption. Because we want to
	// authenticate the cookie name as well as the value, we make this plaintext
	// in the format "{cookie name}:{cookie value}". We use the : character as a
	// separator because it is an invalid character for cookie names and
	// therefore shouldn't appear in them.
	plaintext := fmt.Sprintf("%s:%s", cookie.Name, cookie.Value)

	// Encrypt the data using aesGCM.Seal(). By passing the nonce as the first
	// parameter, the encrypted data will be appended to the nonce — meaning
	// that the returned encryptedValue variable will be in the format
	// "{nonce}{encrypted plaintext data}".
	encryptedValue := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Set the cookie value to the encryptedValue.
	cookie.Value = string(encryptedValue)

	// Write the cookie as normal.
	return c.Write(cookie)
}

func (c *CookieManager) ReadEncrypt(cookie *http.Cookie, name string, secretKey []byte) (string, error) {
	encryptedValue, err := c.Read(cookie)
	log.Println(encryptedValue)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
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

		Secure:   false,
		HttpOnly: false,
	}

	err = gob.NewEncoder(&buffer).Encode(cookie)
	if err != nil {
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

	if len(cookieAll) > 0 {
		for _, cookie := range cookieAll {
			if cookie != nil {
				value, err := c.ReadEncrypt(cookie, cookie.Name, c.Key)

				switch err {
				case ErrCipher:
					err = ErrCipher
				case ErrInvalidValue:
					err = ErrInvalidValue
				case nil:
					return value, nil
				}
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
