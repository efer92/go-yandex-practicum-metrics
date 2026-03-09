package service

import (
	"context"
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

func (s *MetricService) SetOnUpdate(fn func()) {
	s.onUpdate = fn
}

func (s *MetricService) notifyUpdate() {
	if s.onUpdate != nil {
		s.onUpdate()
	}
}

func (s *MetricService) UpdateMetric(ctx context.Context, mType, name, value string) error {
	switch mType {
	case model.Gauge:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		if err := s.storage.UpdateGauge(ctx, name, val); err != nil {
			return err
		}
	case model.Counter:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		if err := s.storage.UpdateCounter(ctx, name, val); err != nil {
			return err
		}
	default:
		return ErrUnknownType
	}
	s.notifyUpdate()
	return nil
}

func (s *MetricService) UpdateMetricFromModel(ctx context.Context, m model.Metrics) error {
	switch m.MType {
	case model.Gauge:
		if m.Value == nil {
			return ErrValueRequired
		}
		if err := s.storage.UpdateGauge(ctx, m.ID, *m.Value); err != nil {
			return fmt.Errorf("failed to update gauge %q: %w", m.ID, err)
		}
	case model.Counter:
		if m.Delta == nil {
			return ErrDeltaRequired
		}
		if err := s.storage.UpdateCounter(ctx, m.ID, *m.Delta); err != nil {
			return fmt.Errorf("failed to update counter %q: %w", m.ID, err)
		}
	default:
		return ErrUnknownType
	}
	s.notifyUpdate()
	return nil
}

func (s *MetricService) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	if err := s.storage.UpdateBatch(ctx, metrics); err != nil {
		return fmt.Errorf("batch update failed: %w", err)
	}
	s.notifyUpdate()
	return nil
}

func (s *MetricService) GetMetric(ctx context.Context, mType, name string) (*model.Metrics, error) {
	if name == "" {
		return nil, ErrMetricNameEmpty
	}
	if mType != model.Gauge && mType != model.Counter {
		return nil, ErrUnknownType
	}
	m, ok := s.storage.GetMetric(ctx, name, mType)
	if !ok {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (s *MetricService) GetValue(ctx context.Context, mType, name string) (string, error) {
	if name == "" {
		return "", ErrMetricNameEmpty
	}
	if mType != model.Gauge && mType != model.Counter {
		return "", ErrUnknownType
	}
	if val, ok := s.storage.GetValue(ctx, mType, name); ok {
		return val, nil
	}
	return "", ErrMetricNotFound
}

func (s *MetricService) GetAllMetrics(ctx context.Context) map[string]string {
	return s.storage.GetAllMetrics(ctx)
}
