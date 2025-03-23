package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// compressibleWriter реализует интерфейс http.ResponseWriter и позволяет
// прозрачно для сервера сжимать передаваемые данные и выставлять правильные
// HTTP-заголовки при наличии подходящего Content-Type
type compressibleWriter struct {
	http.ResponseWriter
	zw       *gzip.Writer
	compress *bool
}

func newCompressWriter(w http.ResponseWriter) *compressibleWriter {
	return &compressibleWriter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
		compress:       nil,
	}
}

func (c *compressibleWriter) Write(p []byte) (int, error) {
	if c.compress == nil {
		compress := slices.Contains(
			[]string{"application/json", "text/html"},
			c.ResponseWriter.Header().Get("Content-Type"))
		if compress {
			c.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		}
		c.compress = &compress
	}
	if *c.compress {
		return c.zw.Write(p)
	} else {
		return c.ResponseWriter.Write(p)
	}
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressibleWriter) Close() error {
	return c.zw.Close()
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	n, err = c.zr.Read(p)
	fmt.Println(string(p))
	return
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func Encoding(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w
		// сжимаем ответ при необходимости
		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		var cw *compressibleWriter
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw = newCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		next.ServeHTTP(ow, r)
	})
}
