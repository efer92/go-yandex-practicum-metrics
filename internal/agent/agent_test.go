package agent

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestCollector_Collect(t *testing.T) {
    c := NewCollector()
    c.Collect()

    metrics := c.metrics

    // Проверяем PollCount
    if metrics.Counter != 1 {
        t.Errorf("Expected Counter=1, got %d", metrics.Counter)
    }

    // Проверяем наличие ключевых метрик
    required := []string{"Alloc", "HeapAlloc", "RandomValue", "TotalAlloc"}
    for _, name := range required {
        if _, ok := metrics.Gauge[name]; !ok {
            t.Errorf("Missing required metric: %s", name)
        }
    }

    // Проверяем что RandomValue в диапазоне [0,1)
    if metrics.Gauge["RandomValue"] < 0 || metrics.Gauge["RandomValue"] >= 1 {
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

    // После GetSnapshot счетчик должен сброситься
    if c.metrics.Counter != 0 {
        t.Error("Counter should be reset after GetSnapshot")
    }
}

func TestSender_SendGauge(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            t.Errorf("Expected POST, got %s", r.Method)
        }
        if r.Header.Get("Content-Type") != "text/plain" {
            t.Error("Expected Content-Type: text/plain")
        }
        expectedPath := "/update/gauge/Alloc/123.456000"
        if r.URL.Path != expectedPath {
            t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    sender := NewSender(server.URL)
    err := sender.SendGauge("Alloc", 123.456)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}

func TestSender_SendCounter(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        expectedPath := "/update/counter/PollCount/5"
        if r.URL.Path != expectedPath {
            t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    sender := NewSender(server.URL)
    err := sender.SendCounter("PollCount", 5)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}

func TestSender_ServerError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusBadRequest)
    }))
    defer server.Close()

    sender := NewSender(server.URL)
    err := sender.SendGauge("test", 1.0)
    if err == nil {
        t.Error("Expected error for status 400")
    }
}

func TestSender_SendBatch(t *testing.T) {
    requestCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestCount++
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    sender := NewSender(server.URL)
    metrics := Metrics{
        Gauge: map[string]float64{
            "metric1": 1.0,
            "metric2": 2.0,
        },
        Counter: 1,
    }

    err := sender.SendBatch(metrics)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }

    // Должно быть 3 запроса: 2 gauge + 1 counter
    if requestCount != 3 {
        t.Errorf("Expected 3 requests, got %d", requestCount)
    }
}

func TestAgent_Report(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    agent := &Agent{
        collector: NewCollector(),
        sender:    NewSender(server.URL),
    }

    agent.collector.Collect()
    
    err := agent.report()
    if err != nil {
        t.Errorf("Report failed: %v", err)
    }
}

func TestSender_SendBatch_Error(t *testing.T) {
    // Падает на первом же запросе
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        if callCount > 1 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    sender := NewSender(server.URL)
    metrics := Metrics{
        Gauge: map[string]float64{
            "metric1": 1.0,
            "metric2": 2.0,
        },
        Counter: 1,
    }

    err := sender.SendBatch(metrics)
    if err == nil {
        t.Error("Expected error when server fails")
    }
}

func TestCollector_MultipleCollects(t *testing.T) {
    c := NewCollector()
    
    // Собираем 5 раз
    for i := 0; i < 5; i++ {
        c.Collect()
    }
    
    if c.metrics.Counter != 5 {
        t.Errorf("Expected Counter=5, got %d", c.metrics.Counter)
    }
    
    // Проверяем что все метрики gauge присутствуют
    expectedMetrics := []string{
        "Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
        "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
        "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
        "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
        "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
        "Sys", "TotalAlloc", "RandomValue",
    }
    
    for _, name := range expectedMetrics {
        if _, ok := c.metrics.Gauge[name]; !ok {
            t.Errorf("Missing metric: %s", name)
        }
    }
}
