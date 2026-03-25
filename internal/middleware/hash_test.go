package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	w.Write(body) //nolint:errcheck
}

func TestHashMiddleware_NoKey_PassesThrough(t *testing.T) {
	handler := HashMiddleware("")(http.HandlerFunc(echoHandler))

	body := `{"id":"Alloc","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get(HeaderHashSHA256) != "" {
		t.Error("expected no HashSHA256 header when key is empty")
	}
}

func TestHashMiddleware_WithKey_SignsResponse(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key)(http.HandlerFunc(echoHandler))

	body := `{"id":"Alloc","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	responseHash := rr.Header().Get(HeaderHashSHA256)
	if responseHash == "" {
		t.Fatal("expected HashSHA256 header in response")
	}

	// Проверяем что хеш ответа корректен
	expected := hash.Sign([]byte(body), key)
	if !hash.Equal(responseHash, expected) {
		t.Errorf("response hash mismatch: got %s, want %s", responseHash, expected)
	}
}

func TestHashMiddleware_WithKey_ValidRequestHash_Passes(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key)(http.HandlerFunc(echoHandler))

	body := `{"id":"PollCount","type":"counter","delta":5}`
	sig := hash.Sign([]byte(body), key)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(HeaderHashSHA256, sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHashMiddleware_WithKey_InvalidRequestHash_Returns400(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key)(http.HandlerFunc(echoHandler))

	body := `{"id":"PollCount","type":"counter","delta":5}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(HeaderHashSHA256, "invalidsignature0000000000000000000000000000000000000000000000000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHashMiddleware_WithKey_NoRequestHash_Passes(t *testing.T) {
	// Заголовок не прислан — сервер не проверяет, просто подписывает ответ
	const key = "secret"
	handler := HashMiddleware(key)(http.HandlerFunc(echoHandler))

	body := `{"id":"Alloc","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get(HeaderHashSHA256) == "" {
		t.Error("expected HashSHA256 header in response even without request hash")
	}
}
