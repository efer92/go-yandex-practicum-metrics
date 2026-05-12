package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileSink is an audit Sink that appends each Event as a JSON line to a file.
type FileSink struct {
	mu   sync.Mutex
	path string
}

// NewFileSink returns a FileSink that writes to path.
func NewFileSink(path string) *FileSink {
	return &FileSink{path: path}
}

// Receive serializes e to JSON and appends it as a new line to the file.
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
