package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
)

type rsaDecoder struct {
	priv *rsa.PrivateKey
}

// RSADecoder возвращает middleware, расшифровывающее тело запроса.
func RSADecoder(privKeyPath string) (func(http.Handler) http.Handler, error) {
	priv, err := loadPrivateKey(privKeyPath)
	if err != nil {
		return nil, err
	}
	d := &rsaDecoder{priv: priv}
	return d.decryptBodyMiddleware, nil
}

func (d *rsaDecoder) decryptBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		encrypted, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}

		decrypted, err := rsa.DecryptPKCS1v15(nil, d.priv, encrypted)
		if err != nil {
			http.Error(w, "decryption failed", http.StatusBadRequest)
			return
		}

		req.Body = io.NopCloser(bytes.NewReader(decrypted))
		req.ContentLength = int64(len(decrypted))
		next.ServeHTTP(w, req)
	})
}
