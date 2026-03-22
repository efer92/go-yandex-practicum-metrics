package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"go.uber.org/zap"
)

// ─── Collector ────────────────────────────────────────────────────────────────

func TestCollector_Collect(t *testing.T) {
	c := NewCollector()
	c.Collect()

	// Используем GetSnapshot вместо прямого доступа к полям
	snapshot := c.GetSnapshot()

	if snapshot.Counter != 1 {
		t.Errorf("Expected Counter=1, got %d", snapshot.Counter)
	}

	required := []string{"Alloc", "HeapAlloc", "RandomValue", "TotalAlloc"}
	for _, name := range required {
		if _, ok := snapshot.Gauge[name]; !ok {
			t.Errorf("Missing required metric: %s", name)
		}
	}

	if snapshot.Gauge["RandomValue"] < 0 || snapshot.Gauge["RandomValue"] >= 1 {
		t.Error("RandomValue out of range [0,1)")
	}
}

func TestCollector_GetSnapshot(t *testing.T) {
	c := NewCollector()
	c.Collect()
	c.Collect() // Counter = 2

	snapshot := c.GetSnapshot()
	if snapshot.Counter != 2 {
		t.Errorf("Expected snapshot Counter=2, got %d", snapshot.Counter)
	}
}

func TestCollector_MultipleCollects(t *testing.T) {
	c := NewCollector()

	for i := 0; i < 5; i++ {
		c.Collect()
	}

	snapshot := c.GetSnapshot()

	if snapshot.Counter != 5 {
		t.Errorf("Expected Counter=5, got %d", snapshot.Counter)
	}

	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range expectedMetrics {
		if _, ok := snapshot.Gauge[name]; !ok {
			t.Errorf("Missing metric: %s", name)
		}
	}
}

func TestCollector_CollectExtra(t *testing.T) {
	c := NewCollector()
	err := c.CollectExtra()
	if err != nil {
		t.Fatalf("CollectExtra() unexpected error: %v", err)
	}

	// Используем GetSnapshot вместо прямого c.mu.Lock()
	snapshot := c.GetSnapshot()

	if _, ok := snapshot.Gauge["TotalMemory"]; !ok {
		t.Error("Missing metric: TotalMemory")
	}
	if _, ok := snapshot.Gauge["FreeMemory"]; !ok {
		t.Error("Missing metric: FreeMemory")
	}
	if _, ok := snapshot.Gauge["CPUutilization1"]; !ok {
		t.Error("Missing metric: CPUutilization1")
	}
}

// ─── Sender ───────────────────────────────────────────────────────────────────

func TestSender_SendGauge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding: gzip, got %s", r.Header.Get("Content-Encoding"))
		}
		if r.URL.Path != "/update" {
			t.Errorf("Expected path /update, got %s", r.URL.Path)
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("Failed to create gzip reader: %v", err)
			return
		}
		defer gr.Close()

		var m model.Metrics
		if err := json.NewDecoder(gr).Decode(&m); err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}
		if m.ID != "Alloc" || m.MType != model.Gauge || m.Value == nil || *m.Value != 123.456 {
			t.Errorf("Unexpected metric: %+v", m)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "")
	val := 123.456
	err := sender.send(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSender_SendCounter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("Failed to create gzip reader: %v", err)
			return
		}
		defer gr.Close()

		var m model.Metrics
		if err := json.NewDecoder(gr).Decode(&m); err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}
		if m.ID != "PollCount" || m.MType != model.Counter || m.Delta == nil || *m.Delta != 5 {
			t.Errorf("Unexpected metric: %+v", m)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "")
	delta := int64(5)
	err := sender.send(model.Metrics{ID: "PollCount", MType: model.Counter, Delta: &delta})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSender_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "")
	val := 1.0
	err := sender.send(model.Metrics{ID: "test", MType: model.Gauge, Value: &val})
	if err == nil {
		t.Error("Expected error for status 400")
	}
}

func TestSender_SendMetrics(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "")

	snapshot := Metrics{
		Gauge:   map[string]float64{"metric1": 1.0, "metric2": 2.0},
		Counter: 1,
	}
	metrics := convertToModelMetrics(snapshot)

	err := sender.SendMetrics(metrics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if got := int(requestCount.Load()); got != 3 {
		t.Errorf("Expected 3 requests, got %d", got)
	}
}

func TestSender_SendMetrics_Error(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) > 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "")

	snapshot := Metrics{
		Gauge:   map[string]float64{"metric1": 1.0, "metric2": 2.0},
		Counter: 1,
	}
	metrics := convertToModelMetrics(snapshot)

	err := sender.SendMetrics(metrics)
	if err == nil {
		t.Error("Expected error when server fails")
	}
}

// ─── Agent ────────────────────────────────────────────────────────────────────

func TestAgent_WorkerSendsBatch(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ag := NewWithConfig(server.URL, 50*time.Millisecond, 100*time.Millisecond, zap.NewNop(), "", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	ag.Run(ctx)

	if received.Load() == 0 {
		t.Error("expected at least one request to be sent")
	}
}

func TestAgent_RateLimitRespected(t *testing.T) {
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := concurrent.Add(1)
		// обновляем максимум атомарно
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		concurrent.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	const limit = 2
	ag := NewWithConfig(server.URL, 10*time.Millisecond, 20*time.Millisecond, zap.NewNop(), "", limit)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ag.Run(ctx)

	if got := maxConcurrent.Load(); got > limit {
		t.Errorf("rate limit exceeded: max concurrent = %d, limit = %d", got, limit)
	}
}

// ─── convertToModelMetrics ────────────────────────────────────────────────────

func TestConvertToModelMetrics(t *testing.T) {
	snapshot := Metrics{
		Gauge:   map[string]float64{"Alloc": 1024.0, "HeapAlloc": 512.0},
		Counter: 7,
	}

	result := convertToModelMetrics(snapshot)

	if len(result) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(result))
	}

	var foundCounter bool
	for _, m := range result {
		if m.MType == model.Counter {
			foundCounter = true
			if m.ID != "PollCount" {
				t.Errorf("Expected PollCount, got %s", m.ID)
			}
			if m.Delta == nil || *m.Delta != 7 {
				t.Errorf("Expected Delta=7, got %v", m.Delta)
			}
		}
		if m.MType == model.Gauge && m.Value == nil {
			t.Errorf("Gauge metric %s has nil Value", m.ID)
		}
	}

	if !foundCounter {
		t.Error("PollCount counter not found in result")
	}
}
