package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
	"go.uber.org/zap"
)

// hashResponseWriter буферизует ответ, чтобы подписать его перед отправкой.
type hashResponseWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *hashResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *hashResponseWriter) flush(key string) {
	body := w.buf.Bytes()
	if key != "" {
		w.ResponseWriter.Header().Set("HashSHA256", hash.Sign(body, key))
	}
	if w.status != 0 {
		w.ResponseWriter.WriteHeader(w.status)
	}
	w.ResponseWriter.Write(body) //nolint:errcheck
}

// HashMiddleware возвращает nil если ключ не задан — middleware не подключается вовсе.
// При наличии ключа проверяет подпись входящего запроса и подписывает ответ.
func HashMiddleware(key string, logger *zap.Logger) func(http.Handler) http.Handler {
	if key == "" {
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read request body", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if incoming := r.Header.Get("HashSHA256"); incoming != "" {
				expected := hash.Sign(body, key)
				if !hash.Equal(incoming, expected) {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
			}

			hrw := &hashResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(hrw, r)
			hrw.flush(key)
		})
	}
}
