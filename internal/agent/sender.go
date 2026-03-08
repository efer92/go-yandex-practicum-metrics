package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/compress"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/retry"
)

type Sender struct {
	serverURL  string
	httpClient *http.Client
}

func NewSender(serverURL string) *Sender {
	return &Sender{
		serverURL:  serverURL,
		httpClient: &http.Client{},
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

	buf, err := compress.GzipData(body)
	if err != nil {
		return fmt.Errorf("compress metric %s: %w", m.ID, err)
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/update", buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

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

	buf, err := compress.GzipData(body)
	if err != nil {
		return fmt.Errorf("compress metrics: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/updates/", buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

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
