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
            return &model.Metrics{
                ID:    name,
                MType: mType,
                Value: &val,
            }, true
        }
    case model.Counter:
        if val, ok := s.counters[name]; ok {
            return &model.Metrics{
                ID:    name,
                MType: mType,
                Delta: &val,
            }, true
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
