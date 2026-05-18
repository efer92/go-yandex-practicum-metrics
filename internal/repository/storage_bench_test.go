package repository

import (
	"context"
	"strconv"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
)

func seedStorage(b *testing.B, s *MemStorage, n int) {
	b.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		_ = s.UpdateGauge(ctx, "gauge"+strconv.Itoa(i), float64(i))
		_ = s.UpdateCounter(ctx, "counter"+strconv.Itoa(i), int64(i))
	}
}

func BenchmarkMemStorage_UpdateGauge(b *testing.B) {
	s := NewMemStorage()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.UpdateGauge(ctx, "Alloc", float64(i))
	}
}

func BenchmarkMemStorage_UpdateBatch(b *testing.B) {
	s := NewMemStorage()
	ctx := context.Background()
	v := 1.5
	d := int64(2)
	batch := []model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &v},
		{ID: "Frees", MType: model.Gauge, Value: &v},
		{ID: "PollCount", MType: model.Counter, Delta: &d},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.UpdateBatch(ctx, batch)
	}
}

func BenchmarkMemStorage_GetAllMetrics(b *testing.B) {
	s := NewMemStorage()
	seedStorage(b, s, 200)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.GetAllMetrics(ctx)
	}
}

func BenchmarkMemStorage_Snapshot(b *testing.B) {
	s := NewMemStorage()
	seedStorage(b, s, 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Snapshot()
	}
}
