package agent

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/compress"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/crypto"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/retry"
)

// Sender posts collected metrics to the server, optionally signing them with HMAC-SHA256
// and optionally encrypting bodies with an RSA public key.
type Sender struct {
	serverURL  string
	httpClient *http.Client
	key        string
	pubKey     *rsa.PublicKey
	localIP    string
}

// NewSender returns a Sender targeted at serverURL.
//   - key enables HMAC-SHA256 signing when non-empty.
//   - pubKey enables RSA-OAEP+AES-GCM encryption of request bodies when non-nil.
//
// The agent's local IPv4 address is detected once and sent in the X-Real-IP
// header so the server can enforce a trusted-subnet policy.
func NewSender(serverURL string, key string, pubKey *rsa.PublicKey) *Sender {
	return &Sender{
		serverURL:  serverURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		key:        key,
		pubKey:     pubKey,
		localIP:    detectLocalIPv4(),
	}
}

// detectLocalIPv4 returns the first non-loopback IPv4 address bound to a local
// interface. An empty string is returned when no such address is found — in
// that case the X-Real-IP header is left unset.
func detectLocalIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}

// isRetriableHTTPError возвращает true для временных сетевых ошибок соединения.
func isRetriableHTTPError(err error) bool {
	var netErr *url.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// newRequest builds a POST request with a gzip-compressed body, an optional
// HashSHA256 signature, and optional RSA-OAEP+AES-GCM encryption when a public
// key is configured.
func (s *Sender) newRequest(url string, body []byte) (*http.Request, error) {
	gz, err := compress.GzipData(body)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	var reqBody io.Reader = gz
	encrypted := false
	if s.pubKey != nil {
		envelope, err := crypto.Encrypt(s.pubKey, gz.Bytes())
		if err != nil {
			return nil, fmt.Errorf("encrypt: %w", err)
		}
		reqBody = bytes.NewReader(envelope)
		encrypted = true
	}

	req, err := http.NewRequest(http.MethodPost, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if encrypted {
		req.Header.Set(httpconst.HeaderCryptoEncrypted, "1")
	}
	if s.localIP != "" {
		req.Header.Set(httpconst.HeaderXRealIP, s.localIP)
	}

	if s.key != "" {
		req.Header.Set(httpconst.HeaderHashSHA256, hash.Sign(body, s.key))
	}

	return req, nil
}

// SendMetrics sends each metric in a separate request to /update.
func (s *Sender) SendMetrics(metrics []model.Metrics) error {
	return retry.Do(func() error {
		for _, m := range metrics {
			if err := s.send(m); err != nil {
				return err
			}
		}
		return nil
	}, isRetriableHTTPError)
}

func (s *Sender) send(m model.Metrics) error {
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	req, err := s.newRequest(s.serverURL+"/update", body)
	if err != nil {
		return fmt.Errorf("build request for metric %s: %w", m.ID, err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send metric %s: %w", m.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status for metric %s: %d", m.ID, resp.StatusCode)
	}

	return nil
}

// SendBatch sends all metrics in a single request to /updates/.
func (s *Sender) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	return retry.Do(func() error {
		return s.sendBatch(metrics)
	}, isRetriableHTTPError)
}

func (s *Sender) sendBatch(metrics []model.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	req, err := s.newRequest(s.serverURL+"/updates/", body)
	if err != nil {
		return fmt.Errorf("build batch request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status for batch: %d", resp.StatusCode)
	}

	return nil
}
