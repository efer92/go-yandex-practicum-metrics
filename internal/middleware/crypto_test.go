package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	cryptopkg "github.com/efer92/go-yandex-practicum-metrics/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func mustGenKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv
}

func TestCryptoMiddleware_NilKeyReturnsNil(t *testing.T) {
	assert.Nil(t, CryptoMiddleware(nil, zap.NewNop()))
}

func TestCryptoMiddleware_DecryptsBody(t *testing.T) {
	priv := mustGenKey(t)
	mw := CryptoMiddleware(priv, zap.NewNop())
	require.NotNil(t, mw)

	plaintext := []byte(`{"id":"Alloc","type":"gauge","value":3.14}`)
	envelope, err := cryptopkg.Encrypt(&priv.PublicKey, plaintext)
	require.NoError(t, err)

	var seen []byte
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = body
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(envelope))
	req.Header.Set(httpconst.HeaderCryptoEncrypted, "1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, plaintext, seen)
}

func TestCryptoMiddleware_PassThroughWithoutHeader(t *testing.T) {
	priv := mustGenKey(t)
	mw := CryptoMiddleware(priv, zap.NewNop())

	plaintext := []byte("not encrypted")
	var seen []byte
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(plaintext))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, plaintext, seen)
}

func TestCryptoMiddleware_BadEnvelopeRejected(t *testing.T) {
	priv := mustGenKey(t)
	mw := CryptoMiddleware(priv, zap.NewNop())

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached")
	}))

	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader([]byte("not really an envelope")))
	req.Header.Set(httpconst.HeaderCryptoEncrypted, "1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
