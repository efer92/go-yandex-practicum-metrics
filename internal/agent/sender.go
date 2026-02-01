package agent

import (
    "fmt"
    "net/http"
    "time"
)

type Sender struct {
    client    *http.Client
    serverURL string
}

func NewSender(serverURL string) *Sender {
    return &Sender{
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
        serverURL: serverURL,
    }
}

// SendGauge отправляет метрику типа gauge
func (s *Sender) SendGauge(name string, value float64) error {
    url := fmt.Sprintf("%s/update/gauge/%s/%f", s.serverURL, name, value)
    return s.send(url)
}

// SendCounter отправляет метрику типа counter
func (s *Sender) SendCounter(name string, value int64) error {
    url := fmt.Sprintf("%s/update/counter/%s/%d", s.serverURL, name, value)
    return s.send(url)
}

func (s *Sender) send(url string) error {
    req, err := http.NewRequest(http.MethodPost, url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "text/plain")

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    return nil
}

// SendBatch отправляет все метрики разом
func (s *Sender) SendBatch(metrics Metrics) error {
    // Отправляем все gauge
    for name, value := range metrics.Gauge {
        if err := s.SendGauge(name, value); err != nil {
            return fmt.Errorf("failed to send gauge %s: %w", name, err)
        }
    }

    // Отправляем счетчик PollCount
    if metrics.Counter > 0 {
        if err := s.SendCounter("PollCount", metrics.Counter); err != nil {
            return fmt.Errorf("failed to send PollCount: %w", err)
        }
    }

    return nil
}