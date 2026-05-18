package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/compress"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/hash"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/retry"
)

const headerHashSHA256 = "HashSHA256"

// Sender posts collected metrics to the server, optionally signing them with HMAC-SHA256.
type Sender struct {
	serverURL  string
	httpClient *http.Client
	key        string
}

// NewSender returns a Sender targeted at serverURL; key enables HMAC-SHA256 signing when non-empty.
func NewSender(serverURL string, key string) *Sender {
	return &Sender{
		serverURL:  serverURL,
		httpClient: &http.Client{},
		key:        key,
	}
}

// isRetriableHTTPError возвращает true для временных сетевых ошибок соединения.
func isRetriableHTTPError(err error) bool {
	var netErr *url.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// newRequest создаёт POST-запрос с gzip-телом и опциональной подписью HashSHA256.
func (s *Sender) newRequest(url string, body []byte) (*http.Request, error) {
	buf, err := compress.GzipData(body)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	// buf — io.Reader после GzipData; нам нужны raw bytes для подписи,
	// поэтому подписываем исходный body (до сжатия), как требует задание:
	// hash считается от тела запроса (JSON), а не от сжатых байт.
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	if s.key != "" {
		req.Header.Set(headerHashSHA256, hash.Sign(body, s.key))
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
