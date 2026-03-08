package repository

import (
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	s := NewMemStorage()

	require.NoError(t, s.UpdateGauge("HeapAlloc", 123.45))

	val, ok := s.GetValue(model.Gauge, "HeapAlloc")
	assert.True(t, ok)
	assert.Equal(t, "123.45", val)

	// Повторная запись замещает значение
	require.NoError(t, s.UpdateGauge("HeapAlloc", 999.99))
	val, _ = s.GetValue(model.Gauge, "HeapAlloc")
	assert.Equal(t, "999.99", val)
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	s := NewMemStorage()

	s.UpdateCounter("PollCount", 10)
	val, ok := s.GetValue(model.Counter, "PollCount")
	assert.True(t, ok)
	assert.Equal(t, "10", val)

	// Счётчик суммируется
	s.UpdateCounter("PollCount", 5)
	val, _ = s.GetValue(model.Counter, "PollCount")
	assert.Equal(t, "15", val)
}

func TestMemStorage_GetValue_NotFound(t *testing.T) {
	s := NewMemStorage()
	_, ok := s.GetValue(model.Gauge, "nonexistent")
	assert.False(t, ok)
}

func TestMemStorage_GetMetric(t *testing.T) {
	s := NewMemStorage()
	s.UpdateGauge("TestGauge", 42.0)

	m, ok := s.GetMetric("TestGauge", model.Gauge)
	require.True(t, ok)
	assert.Equal(t, "TestGauge", m.ID)
	assert.Equal(t, model.Gauge, m.MType)
	require.NotNil(t, m.Value)
	assert.Equal(t, 42.0, *m.Value)
}

func TestMemStorage_GetMetric_NotFound(t *testing.T) {
	s := NewMemStorage()

	_, ok := s.GetMetric("nonexistent", model.Gauge)
	assert.False(t, ok)

	_, ok = s.GetMetric("nonexistent", model.Counter)
	assert.False(t, ok)

	_, ok = s.GetMetric("test", "unknown")
	assert.False(t, ok)
}

func TestMemStorage_Snapshot(t *testing.T) {
	s := NewMemStorage()
	s.UpdateGauge("Alloc", 1.5)
	s.UpdateCounter("PollCount", 3)

	snap := s.Snapshot()
	assert.Len(t, snap, 2)

	byID := make(map[string]model.Metrics, len(snap))
	for _, m := range snap {
		byID[m.ID] = m
	}

	g, ok := byID["Alloc"]
	assert.True(t, ok)
	require.NotNil(t, g.Value)
	assert.Equal(t, 1.5, *g.Value)

	c, ok := byID["PollCount"]
	assert.True(t, ok)
	require.NotNil(t, c.Delta)
	assert.Equal(t, int64(3), *c.Delta)
}

func TestMemStorage_Concurrency(t *testing.T) {
	s := NewMemStorage()
	done := make(chan struct{}, 100)

	for i := 0; i < 100; i++ {
		go func(val int) {
			s.UpdateCounter("counter", 1)
			s.UpdateGauge("gauge", float64(val))
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	val, _ := s.GetValue(model.Counter, "counter")
	assert.Equal(t, "100", val)
}
