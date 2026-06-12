package middleware

import (
	"fmt"
	"net"
	"net/http"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"go.uber.org/zap"
)

// TrustedSubnetMiddleware returns middleware that rejects with 403 any request
// whose X-Real-IP header is absent or falls outside the given CIDR subnet.
// Returns (nil, nil) when cidr is empty so the middleware can be skipped
// entirely. Returns a non-nil error when cidr is set but unparseable.
func TrustedSubnetMiddleware(cidr string, logger *zap.Logger) (func(http.Handler) http.Handler, error) {
	if cidr == "" {
		return nil, nil
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("trusted_subnet: invalid CIDR %q: %w", cidr, err)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get(httpconst.HeaderXRealIP)
			ip := net.ParseIP(raw)
			if ip == nil || !network.Contains(ip) {
				logger.Warn("rejected request from untrusted subnet",
					zap.String("x_real_ip", raw),
					zap.String("trusted_subnet", cidr),
				)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}
