package rsamiddleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
)

type decoder struct {
	priv *rsa.PrivateKey
}

// Decoder возвращает middleware, расшифровывающее тело запроса,
// если Content-Type — application/octet-stream.
func Decoder(privKeyPath string) (func(http.Handler) http.Handler, error) {
	priv, err := loadPrivateKey(privKeyPath)
	if err != nil {
		return nil, err
	}
	d := &decoder{priv: priv}
	return d.decryptBodyMiddleware, nil
}

func (d *decoder) decryptBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/octet-stream" {
			next.ServeHTTP(w, req)
			return
		}

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
		req.Header.Set("Content-Type", "application/json")

		next.ServeHTTP(w, req)
	})
}
