package repository

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"go.uber.org/zap"
)

// FileBackedStorage wraps MemStorage with snapshot persistence to a JSON file.
type FileBackedStorage struct {
	mem    *MemStorage
	path   string
	mu     sync.Mutex
	logger *zap.Logger
}

// NewFileBackedStorage returns a FileBackedStorage that persists snapshots to path.
func NewFileBackedStorage(path string, logger *zap.Logger) *FileBackedStorage {
	return &FileBackedStorage{
		mem:    NewMemStorage(),
		path:   path,
		logger: logger,
	}
}

// UpdateGauge delegates to the wrapped MemStorage.
func (s *FileBackedStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return s.mem.UpdateGauge(ctx, name, value)
}

// UpdateCounter delegates to the wrapped MemStorage.
func (s *FileBackedStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return s.mem.UpdateCounter(ctx, name, value)
}

// UpdateBatch delegates to the wrapped MemStorage.
func (s *FileBackedStorage) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	return s.mem.UpdateBatch(ctx, metrics)
}

// GetMetric delegates to the wrapped MemStorage.
func (s *FileBackedStorage) GetMetric(ctx context.Context, name string, mType string) (*model.Metrics, bool) {
	return s.mem.GetMetric(ctx, name, mType)
}

// GetValue delegates to the wrapped MemStorage.
func (s *FileBackedStorage) GetValue(ctx context.Context, mType string, name string) (string, bool) {
	return s.mem.GetValue(ctx, mType, name)
}

// GetAllMetrics delegates to the wrapped MemStorage.
func (s *FileBackedStorage) GetAllMetrics(ctx context.Context) map[string]string {
	return s.mem.GetAllMetrics(ctx)
}

// Save writes the current snapshot to the configured JSON file.
func (s *FileBackedStorage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.mem.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Load reads a previously saved snapshot from the configured JSON file.
// A missing file is not an error.
func (s *FileBackedStorage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	ctx := context.Background()
	for _, m := range metrics {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				s.mem.UpdateGauge(ctx, m.ID, *m.Value)
			}
		case model.Counter:
			if m.Delta != nil {
				s.mem.UpdateCounter(ctx, m.ID, *m.Delta)
			}
		}
	}
	return nil
}

// StartPeriodicSave spawns a goroutine that calls Save at the given interval until ctx is cancelled.
func (s *FileBackedStorage) StartPeriodicSave(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.Save(); err != nil {
					s.logger.Error("periodic save failed", zap.Error(err))
				} else {
					s.logger.Info("metrics saved", zap.String("path", s.path))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// SaveOnUpdate is a service.SetOnUpdate-compatible callback that logs errors instead of returning them.
func (s *FileBackedStorage) SaveOnUpdate() {
	if err := s.Save(); err != nil {
		s.logger.Error("sync save failed", zap.Error(err))
	}
}

// Close persists the final snapshot — call from a graceful-shutdown path.
func (s *FileBackedStorage) Close() error {
	return s.Save()
}
