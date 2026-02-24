package repository

import (
	"strconv"
	"sync"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
)

type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetMetric(name string, mType string) (*model.Metrics, bool)
	GetValue(mType string, name string) (string, bool)
	GetAllMetrics() map[string]string // ключ: "type/name", значение: string(value)
}

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) UpdateGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return nil
}

func (s *MemStorage) UpdateCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return nil
}

func (s *MemStorage) GetMetric(name string, mType string) (*model.Metrics, bool) {
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

func (s *MemStorage) GetValue(mType string, name string) (string, bool) {
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

func (s *MemStorage) GetAllMetrics() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for name, val := range s.gauges {
		result["gauge/"+name] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	for name, val := range s.counters {
		result["counter/"+name] = strconv.FormatInt(val, 10)
	}
	return result
}

// Snapshot возвращает копию всех метрик — каждый объект управляет только своим мьютексом.
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
