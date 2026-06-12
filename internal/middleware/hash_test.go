package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
	"go.uber.org/zap"
)

func echoHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("echoHandler: failed to read body: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body) //nolint:errcheck
	}
}

func newHashTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

func TestHashMiddleware_NoKey_ReturnsNil(t *testing.T) {
	m := HashMiddleware("", zap.NewNop())
	if m != nil {
		t.Error("expected nil middleware when key is empty")
	}
}

func TestHashMiddleware_WithKey_SignsResponse(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key, newHashTestLogger(t))(echoHandler(t))

	body := `{"id":"Alloc","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	responseHash := rr.Header().Get(httpconst.HeaderHashSHA256)
	if responseHash == "" {
		t.Fatal("expected HashSHA256 header in response")
	}

	expected := hash.Sign([]byte(body), key)
	if !hash.Equal(responseHash, expected) {
		t.Errorf("response hash mismatch: got %s, want %s", responseHash, expected)
	}
}

func TestHashMiddleware_WithKey_ValidRequestHash_Passes(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key, newHashTestLogger(t))(echoHandler(t))

	body := `{"id":"PollCount","type":"counter","delta":5}`
	sig := hash.Sign([]byte(body), key)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(httpconst.HeaderHashSHA256, sig)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHashMiddleware_WithKey_InvalidRequestHash_Returns400(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key, newHashTestLogger(t))(echoHandler(t))

	body := `{"id":"PollCount","type":"counter","delta":5}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(httpconst.HeaderHashSHA256, "0000000000000000000000000000000000000000000000000000000000000000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHashMiddleware_WithKey_NoRequestHash_Passes(t *testing.T) {
	const key = "secret"
	handler := HashMiddleware(key, newHashTestLogger(t))(echoHandler(t))

	body := `{"id":"Alloc","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get(httpconst.HeaderHashSHA256) == "" {
		t.Error("expected HashSHA256 header in response even without request hash")
	}
}
