package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileObserver is an audit Observer that appends each Event as a JSON line to a file.
// The file handle is opened once in the constructor and reused for every write.
type FileObserver struct {
	mu sync.Mutex
	f  *os.File
}

// NewFileObserver opens (or creates) path in append mode and returns a FileObserver.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("audit: open %q: %w", path, err)
	}
	return &FileObserver{f: f}, nil
}

// Receive serializes e to JSON and appends it as a new line to the file.
func (o *FileObserver) Receive(_ context.Context, e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if _, err := o.f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit: write %q: %w", o.f.Name(), err)
	}
	return nil
}

// Close releases the underlying file handle.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.f.Close()
}
