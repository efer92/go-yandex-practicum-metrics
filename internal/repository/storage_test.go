package repository

import (
    "testing"

    "github.com/efer92/go-yandex-practicum-metrics/internal/model"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
    storage := NewMemStorage()

    // Сохраняем значение
    err := storage.UpdateGauge("HeapAlloc", 123.45)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Проверяем что сохранилось
    val, ok := storage.GetValue(model.Gauge, "HeapAlloc")
    if !ok {
        t.Fatal("expected metric to exist")
    }
    if val != "123.45" {
        t.Errorf("expected 123.45, got %s", val)
    }

    // Обновляем - должно заместить
    storage.UpdateGauge("HeapAlloc", 999.99)
    val, _ = storage.GetValue(model.Gauge, "HeapAlloc")
    if val != "999.99" {
        t.Errorf("expected 999.99 after update, got %s", val)
    }
}

func TestMemStorage_UpdateCounter(t *testing.T) {
    storage := NewMemStorage()

    // Первое значение
    storage.UpdateCounter("PollCount", 10)
    val, _ := storage.GetValue(model.Counter, "PollCount")
    if val != "10" {
        t.Errorf("expected 10, got %s", val)
    }

    // Добавляем - должно суммироваться
    storage.UpdateCounter("PollCount", 5)
    val, _ = storage.GetValue(model.Counter, "PollCount")
    if val != "15" {
        t.Errorf("expected 15 (10+5), got %s", val)
    }
}

func TestMemStorage_GetValue_NotFound(t *testing.T) {
    storage := NewMemStorage()

    _, ok := storage.GetValue(model.Gauge, "nonexistent")
    if ok {
        t.Error("expected false for non-existent metric")
    }
}

func TestMemStorage_GetMetric(t *testing.T) {
    storage := NewMemStorage()
    storage.UpdateGauge("TestGauge", 42.0)

    metric, ok := storage.GetMetric("TestGauge", model.Gauge)
    if !ok {
        t.Fatal("expected metric to exist")
    }
    if metric.ID != "TestGauge" {
        t.Errorf("wrong ID: %s", metric.ID)
    }
    if metric.MType != model.Gauge {
        t.Errorf("wrong type: %s", metric.MType)
    }
    if metric.Value == nil || *metric.Value != 42.0 {
        t.Error("wrong value")
    }
}

func TestMemStorage_Concurrency(t *testing.T) {
    storage := NewMemStorage()
    done := make(chan bool)

    // Пишем из нескольких горутин
    for i := 0; i < 100; i++ {
        go func(val int) {
            storage.UpdateCounter("counter", 1)
            storage.UpdateGauge("gauge", float64(val))
            done <- true
        }(i)
    }

    // Ждем завершения
    for i := 0; i < 100; i++ {
        <-done
    }

    // Проверяем что счетчик = 100
    val, _ := storage.GetValue(model.Counter, "counter")
    if val != "100" {
        t.Errorf("expected 100 after concurrent writes, got %s", val)
    }
}

func TestMemStorage_GetMetric_NotFound(t *testing.T) {
    storage := NewMemStorage()
    
    // Пробуем получить несуществующую gauge
    _, ok := storage.GetMetric("nonexistent", model.Gauge)
    if ok {
        t.Error("expected false for non-existent gauge")
    }
    
    // Пробуем получить несуществующую counter
    _, ok = storage.GetMetric("nonexistent", model.Counter)
    if ok {
        t.Error("expected false for non-existent counter")
    }
    
    // Пробуем получить с неверным типом
    _, ok = storage.GetMetric("test", "unknown")
    if ok {
        t.Error("expected false for unknown type")
    }
}
