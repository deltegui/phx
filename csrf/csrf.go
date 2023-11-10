package csrf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CsrfHeaderName string = "X-Csrf-Token"

type Csrf struct {
	cipher cipher.AEAD

	expires time.Duration
}

func NewCsrf(expires time.Duration) Csrf {
	return Csrf{
		cipher:  generateCipher(),
		expires: expires,
	}
}

func generateCipher() cipher.AEAD {
	bytes := make([]byte, 32) //generate a random 32 byte key for AES-256
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalln("Cannot generate random key for aes encryptation in CSRF", err)
	}
	aes, err := aes.NewCipher(bytes)
	if err != nil {
		log.Fatalln("Cannot create cipher for CSRF", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		log.Fatalln("Cannot create CGM:", err)
	}
	return gcm
}

func (csrf *Csrf) encrypt(raw string) string {
	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, csrf.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println("Cannot read from rand:", err)
	}
	dst := csrf.cipher.Seal(nonce, nonce, []byte(raw), nil)
	return base64.RawURLEncoding.EncodeToString(dst)
}

func (csrf *Csrf) decrypt(token string) (string, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("cannot decode base64 csrf token: %s", err)
	}
	nonceSize := csrf.cipher.NonceSize()
	nonce, ciphertext := bytes[:nonceSize], bytes[nonceSize:]
	plaintext, err := csrf.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt csrf token: %s", err)
	}
	return string(plaintext), nil
}

func (csrf Csrf) Generate() string {
	unixTime := time.Now().Unix()
	prime, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		log.Panicln(err)
	}
	raw := fmt.Sprintf("%d//00//%d", prime.Int64(), unixTime)
	e := csrf.encrypt(raw)
	return e
}

func (csrf Csrf) Check(token string) bool {
	raw, err := csrf.decrypt(token)
	if err != nil {
		log.Println("Cannot decrypt csrf token: ", err)
		return false
	}
	fmt.Println(token, "-->", raw)
	parts := strings.Split(raw, "//00//")
	if len(parts) < 2 {
		log.Println("Malformed csrf token. Not enough parts.")
		return false
	}
	unixTime := parts[0]
	i, err := strconv.ParseInt(unixTime, 10, 64)
	if err != nil {
		log.Println("Malformed csrf token. Unixtime is not int64.")
		return false
	}
	t := time.Unix(i, 0)
	if t.After(time.Now().Add(-csrf.expires)) {
		log.Println("Expired csrf token!")
		return false
	}
	return true
}

func (csrf Csrf) CheckRequest(req *http.Request) bool {
	req.ParseForm()
	token := req.Form.Get(CsrfHeaderName)
	if len(token) == 0 {
		log.Printf("Csrf header (%s) token not found\n", CsrfHeaderName)
		return false
	}
	fmt.Println(token)
	return csrf.Check(token)
}
