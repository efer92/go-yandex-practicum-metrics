package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func TestTrustedSubnetMiddleware_EmptyCIDR_SkippedEntirely(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("", zap.NewNop())
	require.NoError(t, err)
	assert.Nil(t, mw)
}

func TestTrustedSubnetMiddleware_InvalidCIDR_ReturnsError(t *testing.T) {
	_, err := TrustedSubnetMiddleware("not-a-cidr", zap.NewNop())
	assert.Error(t, err)
}

func TestTrustedSubnetMiddleware_AllowsInSubnet(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("10.0.0.0/8", zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, mw)

	handler := mw(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set(httpconst.HeaderXRealIP, "10.42.7.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTrustedSubnetMiddleware_Rejects_OutOfSubnet(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("10.0.0.0/8", zap.NewNop())
	require.NoError(t, err)

	handler := mw(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set(httpconst.HeaderXRealIP, "192.168.1.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTrustedSubnetMiddleware_Rejects_MissingHeader(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("10.0.0.0/8", zap.NewNop())
	require.NoError(t, err)

	handler := mw(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTrustedSubnetMiddleware_Rejects_BadIP(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("10.0.0.0/8", zap.NewNop())
	require.NoError(t, err)

	handler := mw(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set(httpconst.HeaderXRealIP, "not-an-ip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
