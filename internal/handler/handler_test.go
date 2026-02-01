package handler

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/efer92/go-yandex-practicum-metrics/internal/repository"
    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

func setupHandler() (*MetricHandler, chi.Router) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)
    return h, h.Routes()
}

func TestUpdateMetric(t *testing.T) {
    _, r := setupHandler()

    tests := []struct {
        name       string
        method     string
        url        string
        wantStatus int
    }{
        {
            name:       "valid counter",
            method:     http.MethodPost,
            url:        "/update/counter/PollCount/10",
            wantStatus: http.StatusOK,
        },
        {
            name:       "valid gauge",
            method:     http.MethodPost,
            url:        "/update/gauge/Alloc/123.45",
            wantStatus: http.StatusOK,
        },
        {
            name:       "empty metric name",
            method:     http.MethodPost,
            url:        "/update/counter//10",
            wantStatus: http.StatusNotFound,
        },
        {
            name:       "invalid metric type",
            method:     http.MethodPost,
            url:        "/update/invalid/test/10",
            wantStatus: http.StatusBadRequest,
        },
        {
            name:       "invalid counter value",
            method:     http.MethodPost,
            url:        "/update/counter/test/3.14",
            wantStatus: http.StatusBadRequest,
        },
        {
            name:       "invalid gauge value",
            method:     http.MethodPost,
            url:        "/update/gauge/test/abc",
            wantStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.url, nil)
            rr := httptest.NewRecorder()

            r.ServeHTTP(rr, req)

            if rr.Code != tt.wantStatus {
                t.Errorf("handler returned wrong status code: got %v want %v",
                    rr.Code, tt.wantStatus)
            }
        })
    }
}

func TestGetValue(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)
    r := h.Routes()

    // Предзаполняем хранилище
    storage.UpdateGauge("HeapAlloc", 999.99)
    storage.UpdateCounter("PollCount", 42)

    tests := []struct {
        name       string
        method     string
        url        string
        wantStatus int
        wantBody   string
    }{
        {
            name:       "get existing gauge",
            method:     http.MethodGet,
            url:        "/value/gauge/HeapAlloc",
            wantStatus: http.StatusOK,
            wantBody:   "999.99",
        },
        {
            name:       "get existing counter",
            method:     http.MethodGet,
            url:        "/value/counter/PollCount",
            wantStatus: http.StatusOK,
            wantBody:   "42",
        },
        {
            name:       "get non-existing metric",
            method:     http.MethodGet,
            url:        "/value/gauge/Missing",
            wantStatus: http.StatusNotFound,
        },
        {
            name:       "empty name",
            method:     http.MethodGet,
            url:        "/value/gauge/",
            wantStatus: http.StatusNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.url, nil)
            rr := httptest.NewRecorder()

            r.ServeHTTP(rr, req)

            if rr.Code != tt.wantStatus {
                t.Errorf("handler returned wrong status code: got %v want %v",
                    rr.Code, tt.wantStatus)
            }

            if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
                t.Errorf("handler returned unexpected body: got %v want %v",
                    rr.Body.String(), tt.wantBody)
            }
        })
    }
}

func TestListMetrics(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)
    r := h.Routes()

    // Добавляем тестовые данные
    storage.UpdateGauge("TestGauge", 100.5)
    storage.UpdateCounter("TestCounter", 42)

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rr := httptest.NewRecorder()

    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rr.Code)
    }

    body := rr.Body.String()
    if !strings.Contains(body, "TestGauge") {
        t.Error("response should contain TestGauge")
    }
    if !strings.Contains(body, "100.5") {
        t.Error("response should contain value 100.5")
    }
	
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("response should contain HTML")
	}

    contentType := rr.Header().Get("Content-Type")
    if !strings.Contains(contentType, "text/html") {
        t.Errorf("Content-Type should contain text/html, got %s", contentType)
    }
}
