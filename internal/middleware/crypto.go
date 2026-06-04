package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/efer92/go-yandex-practicum-metrics/pkg/crypto"
	"go.uber.org/zap"
)

// HeaderCryptoEncrypted is set by the agent on requests whose body is wrapped
// with the asymmetric envelope provided by pkg/crypto.
const HeaderCryptoEncrypted = "X-Crypto-Encrypted"

// CryptoMiddleware returns middleware that decrypts request bodies marked with
// HeaderCryptoEncrypted using priv. Returns nil when priv is nil so the
// middleware can be skipped entirely.
func CryptoMiddleware(priv *rsa.PrivateKey, logger *zap.Logger) func(http.Handler) http.Handler {
	if priv == nil {
		return nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(HeaderCryptoEncrypted) == "" {
				next.ServeHTTP(w, r)
				return
			}
			envelope, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error("crypto: read body", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			plaintext, err := crypto.Decrypt(priv, envelope)
			if err != nil {
				logger.Error("crypto: decrypt", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))
			r.Header.Del(HeaderCryptoEncrypted)
			next.ServeHTTP(w, r)
		})
	}
}
