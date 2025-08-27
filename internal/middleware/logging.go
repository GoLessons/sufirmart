package middleware

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

func NewLoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &loggingDecorator{ResponseWriter: w}

			next.ServeHTTP(rw, r)

			// Логируем информацию о запросе и ответе
			logger.Info(fmt.Sprintf("%s %s", r.Method, r.RequestURI),
				zap.Any("request_headers", r.Header),
				zap.Int("status", rw.statusCode),
				zap.Any("response_headers", rw.headers),
				zap.String("response_body", rw.body),
				zap.Int("response_size", rw.size),
			)
		})
	}
}

type loggingDecorator struct {
	http.ResponseWriter
	statusCode int
	size       int
	body       string
	headers    http.Header
}

func (rw *loggingDecorator) WriteHeader(code int) {
	rw.statusCode = code
	rw.headers = cloneHeader(rw.Header())
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *loggingDecorator) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	rw.body = string(b)
	return size, err
}

func cloneHeader(h http.Header) http.Header {
	c := make(http.Header, len(h))
	for k, vv := range h {
		c[k] = append([]string(nil), vv...)
	}
	return c
}
