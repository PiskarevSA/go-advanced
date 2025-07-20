package rsamiddleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
)

type encoder struct {
	pub *rsa.PublicKey
}

// Encoder возвращает функцию, шифрующую тело запроса с использованием RSA.
func Encoder(pubKeyPath string) (func(*http.Request) error, error) {
	pub, err := loadPublicKey(pubKeyPath)
	if err != nil {
		return nil, err
	}
	e := &encoder{pub: pub}
	return e.encryptBody, nil
}

func (e *encoder) encryptBody(req *http.Request) error {
	if req.Body == nil {
		return nil
	}
	defer req.Body.Close()

	plain, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, e.pub, plain)
	if err != nil {
		return err
	}

	req.Body = io.NopCloser(bytes.NewReader(encrypted))
	req.ContentLength = int64(len(encrypted))
	req.Header.Set("Content-Type", "application/octet-stream")
	return nil
}
