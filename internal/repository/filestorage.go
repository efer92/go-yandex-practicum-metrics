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

// FileBackedStorage — обёртка над MemStorage с поддержкой сохранения на диск.
type FileBackedStorage struct {
	mem    *MemStorage
	path   string
	mu     sync.Mutex
	logger *zap.Logger
}

func NewFileBackedStorage(path string, logger *zap.Logger) *FileBackedStorage {
	return &FileBackedStorage{
		mem:    NewMemStorage(),
		path:   path,
		logger: logger,
	}
}

// --- реализация интерфейса Storage (делегируем в MemStorage) ---

func (s *FileBackedStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return s.mem.UpdateGauge(ctx, name, value)
}

func (s *FileBackedStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return s.mem.UpdateCounter(ctx, name, value)
}

func (s *FileBackedStorage) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	return s.mem.UpdateBatch(ctx, metrics)
}

func (s *FileBackedStorage) GetMetric(ctx context.Context, name string, mType string) (*model.Metrics, bool) {
	return s.mem.GetMetric(ctx, name, mType)
}

func (s *FileBackedStorage) GetValue(ctx context.Context, mType string, name string) (string, bool) {
	return s.mem.GetValue(ctx, mType, name)
}

func (s *FileBackedStorage) GetAllMetrics(ctx context.Context) map[string]string {
	return s.mem.GetAllMetrics(ctx)
}

// --- файловые операции ---

// Save сохраняет все метрики в JSON-файл.
func (s *FileBackedStorage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.mem.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Load загружает метрики из JSON-файла.
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

// StartPeriodicSave запускает фоновое периодическое сохранение с заданным интервалом.
// Останавливается когда ctx отменён.
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

// SaveOnUpdate — колбэк для service.SetOnUpdate.
// Логирует ошибку вместо её возврата, т.к. колбэк не возвращает error.
func (s *FileBackedStorage) SaveOnUpdate() {
	if err := s.Save(); err != nil {
		s.logger.Error("sync save failed", zap.Error(err))
	}
}

// Close сохраняет метрики при завершении — вызывать при graceful shutdown.
func (s *FileBackedStorage) Close() error {
	return s.Save()
}
