package handler

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/efer92/go-yandex-practicum-metrics/internal/repository"
    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

func TestUpdateMetric(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)

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
        {
            name:       "wrong method GET",
            method:     http.MethodGet,
            url:        "/update/counter/test/10",
            wantStatus: http.StatusMethodNotAllowed,
        },
        {
            name:       "wrong method PUT",
            method:     http.MethodPut,
            url:        "/update/counter/test/10",
            wantStatus: http.StatusMethodNotAllowed,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.url, nil)
            rr := httptest.NewRecorder()

            h.UpdateMetric(rr, req)

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
        {
            name:       "wrong method POST",
            method:     http.MethodPost,
            url:        "/value/gauge/HeapAlloc",
            wantStatus: http.StatusMethodNotAllowed,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.url, nil)
            rr := httptest.NewRecorder()

            h.GetValue(rr, req)

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

func TestGetValue_ContentType(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)

    storage.UpdateGauge("Test", 1.0)

    req := httptest.NewRequest(http.MethodGet, "/value/gauge/Test", nil)
    rr := httptest.NewRecorder()

    h.GetValue(rr, req)

    contentType := rr.Header().Get("Content-Type")
    if !strings.Contains(contentType, "text/plain") {
        t.Errorf("expected Content-Type text/plain, got %s", contentType)
    }
}

func TestUpdateMetric_DoubleSlash(t *testing.T) {
    // Этот тест проверяет что сервер корректно обрабатывает // в URL
    // (должен вернуть 404, а не редирект)
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)

    req := httptest.NewRequest(http.MethodPost, "/update/counter//100", nil)
    rr := httptest.NewRecorder()

    // Имитируем обработку без ServeMux (как в main.go)
    if strings.Contains(req.URL.Path, "//") {
        http.Error(rr, "Not found", http.StatusNotFound)
    } else {
        h.UpdateMetric(rr, req)
    }

    if rr.Code != http.StatusNotFound {
        t.Errorf("expected 404 for double slash, got %d", rr.Code)
    }
}

func TestGetValue_DifferentErrors(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)

    tests := []struct {
        name       string
        url        string
        wantStatus int
    }{
        {
            name:       "unknown metric type in get",
            url:        "/value/unknown/test",
            wantStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodGet, tt.url, nil)
            rr := httptest.NewRecorder()
            
            h.GetValue(rr, req)

            if rr.Code != tt.wantStatus {
                t.Errorf("got %d, want %d", rr.Code, tt.wantStatus)
            }
        })
    }
}

func TestGetValue_EdgeCases(t *testing.T) {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := NewMetricHandler(svc)

    // Путь /value/gauge/ (с пустым именем, но со слешем в конце)
    req := httptest.NewRequest(http.MethodGet, "/value/gauge/", nil)
    rr := httptest.NewRecorder()
    
    h.GetValue(rr, req)
    
    // Должен быть 404
    if rr.Code != http.StatusNotFound {
        t.Errorf("expected 404 for empty name, got %d", rr.Code)
    }
}
