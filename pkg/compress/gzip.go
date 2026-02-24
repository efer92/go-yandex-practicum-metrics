package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

// GzipData сжимает данные и возвращает буфер с результатом.
// Используется агентом при отправке запросов на сервер.
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
