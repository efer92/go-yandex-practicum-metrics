// Package middleware contains HTTP middleware: structured request logging,
// gzip compression of request/response bodies, and HMAC-SHA256 signing.
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter — обёртка над http.ResponseWriter для перехвата кода статуса и размера ответа
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Logger returns middleware that logs each request and response through zap.
func Logger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // дефолт, если WriteHeader не вызван явно
			}

			next.ServeHTTP(rw, r)

			log.Info("request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", rw.status),
				zap.Int("size", rw.size),
			)
		})
	}
}
