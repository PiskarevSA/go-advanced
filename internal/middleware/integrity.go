package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
)

const integrityKey = "HashSHA256"

type signedWriter struct {
	http.ResponseWriter
	bodyBuffer bytes.Buffer
	signer     hash.Hash
}

func newSignedWriter(w http.ResponseWriter, key string) *signedWriter {
	return &signedWriter{
		ResponseWriter: w,
		bodyBuffer:     *bytes.NewBuffer(nil),
		signer:         hmac.New(sha256.New, []byte(key)),
	}
}

func (w *signedWriter) Write(p []byte) (int, error) {
	if _, err := w.signer.Write(p); err != nil {
		return 0, fmt.Errorf("signer: %w", err)
	}
	// delay actual writing until hash will be calculated
	return w.bodyBuffer.Write(p)
}

func (w *signedWriter) Sign() error {
	sign := w.signer.Sum(nil)
	hexSum := hex.EncodeToString(sign[:])
	w.ResponseWriter.Header().Set("HashSHA256", hexSum)
	// do actual writing
	if _, err := w.ResponseWriter.Write(w.bodyBuffer.Bytes()); err != nil {
		return fmt.Errorf("signer: %w", err)
	}
	return nil
}

// SignVerifier verifies header "HashSHA256" using provided key
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
		if !c.verifyRequest(w, r) {
			return
		}

		sw := newSignedWriter(w, c.key)

		next.ServeHTTP(sw, r)

		c.signResponse(sw)
	})
}

func (c *SignVerifier) verifyRequest(w http.ResponseWriter, r *http.Request) bool {
	expectedHexSum := r.Header.Values(integrityKey)
	if len(expectedHexSum) == 0 {
		return true
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	defer func() {
		// restore body for future read
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	h := hmac.New(sha256.New, []byte(c.key))
	h.Write(body)
	sign := h.Sum(nil)
	actualHexSum := hex.EncodeToString(sign[:])
	if actualHexSum != expectedHexSum[0] {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return false
	}

	return true
}

func (c *SignVerifier) signResponse(sw *signedWriter) {
	if err := sw.Sign(); err != nil {
		http.Error(sw, err.Error(), http.StatusInternalServerError)
	}
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
