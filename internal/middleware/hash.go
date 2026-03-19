package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
)

const HeaderHashSHA256 = "HashSHA256"

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
		w.ResponseWriter.Header().Set(HeaderHashSHA256, hash.Sign(body, key))
	}
	if w.status != 0 {
		w.ResponseWriter.WriteHeader(w.status)
	}
	w.ResponseWriter.Write(body) //nolint:errcheck
}

// HashMiddleware проверяет подпись входящего запроса и подписывает ответ
func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if incoming := r.Header.Get(HeaderHashSHA256); incoming != "" {
				expected := hash.Sign(body, key)
				if !hash.Equal(incoming, expected) {
					http.Error(w, "invalid hash", http.StatusBadRequest)
					return
				}
			}

			hrw := &hashResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(hrw, r)
			hrw.flush(key)
		})
	}
}
