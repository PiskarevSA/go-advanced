package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

const integrityKey = "HashSHA256"

// SignVerifier verifies header "HashSHA256" and does nothing if key is empty
type SignVerifier struct {
	key string
}

func NewSignVerifier(key string) *SignVerifier {
	return &SignVerifier{
		key: key,
	}
}

func (c *SignVerifier) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify request if hash provided
		if expectedHexSum := r.Header.Values(integrityKey); len(expectedHexSum) > 0 {
			// read body
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// check integrity
			h := hmac.New(sha256.New, []byte(c.key))
			h.Write(body)
			sign := h.Sum(nil)
			actualHexSum := hex.EncodeToString(sign[:])
			if actualHexSum != expectedHexSum[0] {
				http.Error(w, "invalid signature", http.StatusBadRequest)
				return
			}

			// restore body for future read
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		next.ServeHTTP(w, r)
	})
}

func Integrity(key string) func(next http.Handler) http.Handler {
	if len(key) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	v := NewSignVerifier(key)
	return v.Handler
}
