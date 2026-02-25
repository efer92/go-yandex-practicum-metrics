package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
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

func (s *Sender) SendBatch(metrics []model.Metrics) error {
	for _, m := range metrics {
		if err := s.send(m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sender) send(m model.Metrics) error {
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.serverURL+"/update",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("send metric %s: %w", m.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status for metric %s: %d", m.ID, resp.StatusCode)
	}

	return nil
}
