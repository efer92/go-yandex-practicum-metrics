// Package repository provides metric storage backends behind a common interface.
package repository

import (
	"context"
	"strconv"
	"sync"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
)

// Storage is the persistence contract implemented by every backend (in-memory, file-backed, database).
type Storage interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateBatch(ctx context.Context, metrics []model.Metrics) error
	GetMetric(ctx context.Context, name string, mType string) (*model.Metrics, bool)
	GetValue(ctx context.Context, mType string, name string) (string, bool)
	GetAllMetrics(ctx context.Context) map[string]string
}

// MemStorage is the in-memory Storage implementation, safe for concurrent use.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage returns an empty MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge sets the gauge value, replacing any previous one.
func (s *MemStorage) UpdateGauge(_ context.Context, name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return nil
}

// UpdateCounter adds the delta to the current counter value.
func (s *MemStorage) UpdateCounter(_ context.Context, name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return nil
}

// UpdateBatch applies a slice of mixed gauge/counter updates atomically.
func (s *MemStorage) UpdateBatch(_ context.Context, metrics []model.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				s.gauges[m.ID] = *m.Value
			}
		case model.Counter:
			if m.Delta != nil {
				s.counters[m.ID] += *m.Delta
			}
		}
	}
	return nil
}

// GetMetric returns the metric and true, or nil and false if it is absent.
func (s *MemStorage) GetMetric(_ context.Context, name string, mType string) (*model.Metrics, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch mType {
	case model.Gauge:
		if val, ok := s.gauges[name]; ok {
			return &model.Metrics{ID: name, MType: mType, Value: &val}, true
		}
	case model.Counter:
		if val, ok := s.counters[name]; ok {
			return &model.Metrics{ID: name, MType: mType, Delta: &val}, true
		}
	}
	return nil, false
}

// GetValue returns the metric value formatted as text and true, or "" and false if it is absent.
func (s *MemStorage) GetValue(_ context.Context, mType string, name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch mType {
	case model.Gauge:
		if val, ok := s.gauges[name]; ok {
			return strconv.FormatFloat(val, 'f', -1, 64), true
		}
	case model.Counter:
		if val, ok := s.counters[name]; ok {
			return strconv.FormatInt(val, 10), true
		}
	}
	return "", false
}

// GetAllMetrics returns every metric formatted as "<type>/<name>" → value.
func (s *MemStorage) GetAllMetrics(_ context.Context) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string, len(s.gauges)+len(s.counters))
	for name, val := range s.gauges {
		result["gauge/"+name] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	for name, val := range s.counters {
		result["counter/"+name] = strconv.FormatInt(val, 10)
	}
	return result
}

// Snapshot returns a typed copy of all metrics, suitable for serialization.
func (s *MemStorage) Snapshot() []model.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := make([]model.Metrics, 0, len(s.gauges)+len(s.counters))
	for name, val := range s.gauges {
		v := val
		metrics = append(metrics, model.Metrics{ID: name, MType: model.Gauge, Value: &v})
	}
	for name, val := range s.counters {
		d := val
		metrics = append(metrics, model.Metrics{ID: name, MType: model.Counter, Delta: &d})
	}
	return metrics
}
