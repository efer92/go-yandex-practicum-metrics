package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPObserver is an audit Observer that POSTs each Event as JSON to a remote URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver returns an HTTPObserver targeted at url with a 5-second client timeout.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Receive POSTs the JSON-encoded Event to the configured URL.
// Any non-2xx response is treated as an error.
func (o *HTTPObserver) Receive(ctx context.Context, e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("audit: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("audit: post %q: %w", o.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit: unexpected status %d from %q", resp.StatusCode, o.url)
	}
	return nil
}
