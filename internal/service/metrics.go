package service

import (
	"errors"
	"strconv"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
)

type MetricService struct {
	storage repository.Storage
}

func NewMetricService(storage repository.Storage) *MetricService {
	return &MetricService{storage: storage}
}

// UpdateMetric — старый метод для text/plain эндпоинта
func (s *MetricService) UpdateMetric(mType, name, value string) error {
	switch mType {
	case model.Gauge:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("invalid gauge value")
		}
		return s.storage.UpdateGauge(name, val)

	case model.Counter:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("invalid counter value")
		}
		return s.storage.UpdateCounter(name, val)

	default:
		return errors.New("unknown metric type")
	}
}

// UpdateMetricFromModel — новый метод для JSON эндпоинта
func (s *MetricService) UpdateMetricFromModel(m model.Metrics) error {
	switch m.MType {
	case model.Gauge:
		if m.Value == nil {
			return errors.New("value is required for gauge")
		}
		return s.storage.UpdateGauge(m.ID, *m.Value)

	case model.Counter:
		if m.Delta == nil {
			return errors.New("delta is required for counter")
		}
		return s.storage.UpdateCounter(m.ID, *m.Delta)

	default:
		return errors.New("unknown metric type")
	}
}

// GetMetric — возвращает метрику как model.Metrics для JSON эндпоинта
func (s *MetricService) GetMetric(mType, name string) (*model.Metrics, error) {
	if name == "" {
		return nil, errors.New("metric name is empty")
	}
	if mType != model.Gauge && mType != model.Counter {
		return nil, errors.New("unknown metric type")
	}

	m, ok := s.storage.GetMetric(name, mType)
	if !ok {
		return nil, errors.New("metric not found")
	}
	return m, nil
}

// GetValue — старый метод для text/plain эндпоинта
func (s *MetricService) GetValue(mType, name string) (string, error) {
	if name == "" {
		return "", errors.New("metric name is empty")
	}
	if mType != model.Gauge && mType != model.Counter {
		return "", errors.New("unknown metric type")
	}

	if val, ok := s.storage.GetValue(mType, name); ok {
		return val, nil
	}
	return "", errors.New("metric not found")
}

func (s *MetricService) GetAllMetrics() map[string]string {
	return s.storage.GetAllMetrics()
}
