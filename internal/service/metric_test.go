package service

import (
	"context"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
)

func TestMetricService_UpdateMetric(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	tests := []struct {
		name    string
		mType   string
		mName   string
		value   string
		wantErr bool
	}{
		{
			name:    "valid counter",
			mType:   model.Counter,
			mName:   "PollCount",
			value:   "10",
			wantErr: false,
		},
		{
			name:    "valid gauge",
			mType:   model.Gauge,
			mName:   "HeapAlloc",
			value:   "123.45",
			wantErr: false,
		},
		{
			name:    "invalid type",
			mType:   "unknown",
			mName:   "test",
			value:   "10",
			wantErr: true,
		},
		{
			name:    "invalid counter value",
			mType:   model.Counter,
			mName:   "test",
			value:   "3.14",
			wantErr: true,
		},
		{
			name:    "invalid gauge value",
			mType:   model.Gauge,
			mName:   "test",
			value:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateMetric(ctx, tt.mType, tt.mName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricService_UpdateMetric_Logic(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	// Counter должен суммироваться
	svc.UpdateMetric(ctx, model.Counter, "count", "10")
	svc.UpdateMetric(ctx, model.Counter, "count", "5")
	val, _ := storage.GetValue(ctx, model.Counter, "count")
	if val != "15" {
		t.Errorf("expected counter 15, got %s", val)
	}

	// Gauge должен замещаться
	svc.UpdateMetric(ctx, model.Gauge, "gauge", "10.5")
	svc.UpdateMetric(ctx, model.Gauge, "gauge", "20.5")
	val, _ = storage.GetValue(ctx, model.Gauge, "gauge")
	if val != "20.5" {
		t.Errorf("expected gauge 20.5, got %s", val)
	}
}

func TestMetricService_GetValue(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	// Сохраняем метрику
	storage.UpdateGauge(ctx, "Alloc", 100.5)

	tests := []struct {
		name    string
		mType   string
		mName   string
		want    string
		wantErr bool
	}{
		{
			name:    "existing metric",
			mType:   model.Gauge,
			mName:   "Alloc",
			want:    "100.5",
			wantErr: false,
		},
		{
			name:    "non-existing metric",
			mType:   model.Gauge,
			mName:   "Missing",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty name",
			mType:   model.Gauge,
			mName:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid type",
			mType:   "invalid",
			mName:   "test",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetValue(ctx, tt.mType, tt.mName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
