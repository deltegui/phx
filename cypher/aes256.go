package cypher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"github.com/deltegui/phx/core"
)

type AES256 struct {
	cipher cipher.AEAD
}

func GenerateRandomPass() []byte {
	bytes := make([]byte, 32) //generate a random 32 byte key for AES-256
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalln("Cannot generate random key for aes encryptation in CSRF", err)
	}
	return bytes
}

func GenerateRandomPassAsString() string {
	return base64.RawStdEncoding.EncodeToString(GenerateRandomPass())
}

func generateCipher(pass []byte) cipher.AEAD {
	if len(pass) != 32 {
		log.Fatalln("The csrf encrypt password must be 32 bit long")
	}
	aes, err := aes.NewCipher(pass)
	if err != nil {
		log.Fatalln("Cannot create cipher for CSRF", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		log.Fatalln("Cannot create CGM:", err)
	}
	return gcm
}

func New() core.Cypher {
	return AES256{
		cipher: generateCipher(GenerateRandomPass()),
	}
}

func NewWithPassword(password []byte) core.Cypher {
	return AES256{
		cipher: generateCipher(password),
	}
}

func NewWithPasswordAsString(password string) core.Cypher {
	bytes, err := base64.RawStdEncoding.DecodeString(password)
	if err != nil {
		log.Panicln("Cannot decode password for cypher:", err)
	}
	return NewWithPassword(bytes)
}

func (aes AES256) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, aes.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("cannot read from rand: %s", err)
	}
	dst := aes.cipher.Seal(nonce, nonce, data, nil)
	return dst, nil
}

func (aes AES256) Decrypt(data []byte) ([]byte, error) {
	nonceSize := aes.cipher.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("malformed csrf token")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aes.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt csrf token: %s", err)
	}
	return plaintext, nil
}

func EncodeCookie(cypher core.Cypher, data string) (string, error) {
	bytes, err := cypher.Encrypt([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func DecodeCookie(cypher core.Cypher, data string) (string, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	plaintext, err := cypher.Decrypt(bytes)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
