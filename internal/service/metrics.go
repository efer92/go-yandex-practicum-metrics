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
