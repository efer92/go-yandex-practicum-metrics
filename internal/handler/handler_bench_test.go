package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func benchHandler(b *testing.B) http.Handler {
	b.Helper()
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := NewMetricHandler(svc, zap.NewNop(), nil, nil)
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return r
}

func seedHandler(b *testing.B, r http.Handler, n int) {
	b.Helper()
	v := 1.0
	d := int64(1)
	batch := make([]model.Metrics, 0, 2*n)
	for i := 0; i < n; i++ {
		batch = append(batch, model.Metrics{ID: "gauge" + strconv.Itoa(i), MType: model.Gauge, Value: &v})
		batch = append(batch, model.Metrics{ID: "counter" + strconv.Itoa(i), MType: model.Counter, Delta: &d})
	}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
}

func BenchmarkHandler_UpdateMetricJSON(b *testing.B) {
	r := benchHandler(b)
	val := 42.5
	body, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}

func BenchmarkHandler_UpdateMetricsBatch(b *testing.B) {
	r := benchHandler(b)
	v := 1.0
	d := int64(1)
	batch := make([]model.Metrics, 0, 60)
	for i := 0; i < 30; i++ {
		batch = append(batch, model.Metrics{ID: "gauge" + strconv.Itoa(i), MType: model.Gauge, Value: &v})
		batch = append(batch, model.Metrics{ID: "counter" + strconv.Itoa(i), MType: model.Counter, Delta: &d})
	}
	body, _ := json.Marshal(batch)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}

func BenchmarkHandler_ListMetrics(b *testing.B) {
	r := benchHandler(b)
	seedHandler(b, r, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}

func BenchmarkHandler_GetValueJSON(b *testing.B) {
	r := benchHandler(b)
	seedHandler(b, r, 100)
	body, _ := json.Marshal(model.Metrics{ID: "gauge42", MType: model.Gauge})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}
