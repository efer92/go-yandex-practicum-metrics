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

func (s *FileBackedStorage) UpdateGauge(name string, value float64) error {
	return s.mem.UpdateGauge(name, value)
}

func (s *FileBackedStorage) UpdateCounter(name string, value int64) error {
	return s.mem.UpdateCounter(name, value)
}

func (s *FileBackedStorage) GetMetric(name string, mType string) (*model.Metrics, bool) {
	return s.mem.GetMetric(name, mType)
}

func (s *FileBackedStorage) GetValue(mType string, name string) (string, bool) {
	return s.mem.GetValue(mType, name)
}

func (s *FileBackedStorage) GetAllMetrics() map[string]string {
	return s.mem.GetAllMetrics()
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

	for _, m := range metrics {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				s.mem.UpdateGauge(m.ID, *m.Value)
			}
		case model.Counter:
			if m.Delta != nil {
				s.mem.UpdateCounter(m.ID, *m.Delta)
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
