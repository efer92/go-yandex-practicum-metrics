// Package compress provides gzip helpers used by the agent's HTTP client.
package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

// GzipData compresses data with gzip.BestSpeed and returns the resulting buffer.
func GzipData(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err = gz.Write(data); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err = gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return &buf, nil
}
