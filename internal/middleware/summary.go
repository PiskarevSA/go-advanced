package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	responseStatusCode int
	responseSize       int
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseSize += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseStatusCode = statusCode
}

func Summary(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := loggingResponseWriter{
			ResponseWriter:     w,
			responseStatusCode: http.StatusOK, // WriteHeader() may not be called
			responseSize:       0,
		}
		next.ServeHTTP(&lw, r)
		slog.Info("summary",
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", time.Since(start),
			"status", lw.responseStatusCode,
			"size", lw.responseSize,
		)
	})
}
