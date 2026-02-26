package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipResponseWriter перехватывает запись и сжимает её
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer      io.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipMiddleware — сжимает ответ если клиент поддерживает gzip,
// и распаковывает тело запроса если оно сжато.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковываем входящий запрос если Content-Encoding: gzip
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gr.Close()
			r.Body = io.NopCloser(gr)
		}

		// Сжимаем ответ если клиент принимает gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, "failed to create gzip writer", http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		grw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}

		// Устанавливаем заголовок только для сжимаемых типов контента.
		// Так как Content-Type может быть выставлен хендлером позже,
		// оборачиваем Write чтобы выставить заголовок в нужный момент.
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // размер изменится после сжатия

		next.ServeHTTP(grw, r)
	})
}
