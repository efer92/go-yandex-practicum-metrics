package service

import (
	"fmt"
	"strconv"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
)

type MetricService struct {
	storage  repository.Storage
	onUpdate func()
}

func NewMetricService(storage repository.Storage) *MetricService {
	return &MetricService{storage: storage}
}

// SetOnUpdate регистрирует колбэк, вызываемый после каждого обновления метрики.
// Используется для синхронной записи на диск при StoreInterval == 0.
func (s *MetricService) SetOnUpdate(fn func()) {
	s.onUpdate = fn
}

func (s *MetricService) notifyUpdate() {
	if s.onUpdate != nil {
		s.onUpdate()
	}
}

// UpdateMetric — старый метод для text/plain эндпоинта
func (s *MetricService) UpdateMetric(mType, name, value string) error {
	switch mType {
	case model.Gauge:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		if err := s.storage.UpdateGauge(name, val); err != nil {
			return err
		}
	case model.Counter:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		if err := s.storage.UpdateCounter(name, val); err != nil {
			return err
		}
	default:
		return ErrUnknownType
	}
	s.notifyUpdate()
	return nil
}

// UpdateMetricFromModel — новый метод для JSON эндпоинта
func (s *MetricService) UpdateMetricFromModel(m model.Metrics) error {
	switch m.MType {
	case model.Gauge:
		if m.Value == nil {
			return ErrValueRequired
		}
		if err := s.storage.UpdateGauge(m.ID, *m.Value); err != nil {
			return fmt.Errorf("failed to update gauge %q: %w", m.ID, err)
		}
	case model.Counter:
		if m.Delta == nil {
			return ErrDeltaRequired
		}
		if err := s.storage.UpdateCounter(m.ID, *m.Delta); err != nil {
			return fmt.Errorf("failed to update counter %q: %w", m.ID, err)
		}
	default:
		return ErrUnknownType
	}
	s.notifyUpdate()
	return nil
}

// GetMetric — возвращает метрику как model.Metrics для JSON эндпоинта
func (s *MetricService) GetMetric(mType, name string) (*model.Metrics, error) {
	if name == "" {
		return nil, ErrMetricNameEmpty
	}
	if mType != model.Gauge && mType != model.Counter {
		return nil, ErrUnknownType
	}
	m, ok := s.storage.GetMetric(name, mType)
	if !ok {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

// GetValue — старый метод для text/plain эндпоинта
func (s *MetricService) GetValue(mType, name string) (string, error) {
	if name == "" {
		return "", ErrMetricNameEmpty
	}
	if mType != model.Gauge && mType != model.Counter {
		return "", ErrUnknownType
	}
	if val, ok := s.storage.GetValue(mType, name); ok {
		return val, nil
	}
	return "", ErrMetricNotFound
}

func (s *MetricService) GetAllMetrics() map[string]string {
	return s.storage.GetAllMetrics()
}
