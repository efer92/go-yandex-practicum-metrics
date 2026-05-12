package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type FileSink struct {
	mu   sync.Mutex
	path string
}

func NewFileSink(path string) *FileSink {
	return &FileSink{path: path}
}

func (s *FileSink) Receive(_ context.Context, e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("audit: open %q: %w", s.path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit: write %q: %w", s.path, err)
	}
	return nil
}
