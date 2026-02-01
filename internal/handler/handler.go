package handler

import (
    "net/http"
    "strings"

    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

type MetricHandler struct {
    svc *service.MetricService
}

func NewMetricHandler(svc *service.MetricService) *MetricHandler {
    return &MetricHandler{svc: svc}
}

func (h *MetricHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    path := strings.Trim(r.URL.Path, "/")
    parts := strings.Split(path, "/")

    if len(parts) != 4 || parts[0] != "update" {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    mType := parts[1]
    name := parts[2]
    value := parts[3]

    if name == "" {
        http.NotFound(w, r)
        return
    }

    if err := h.svc.UpdateMetric(mType, name, value); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) GetValue(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    path := strings.Trim(r.URL.Path, "/")
    parts := strings.Split(path, "/")

    if len(parts) != 3 || parts[0] != "value" {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    mType := parts[1]
    name := parts[2]

    if name == "" {
        http.NotFound(w, r)
        return
    }

    val, err := h.svc.GetValue(mType, name)
    if err != nil {
        if err.Error() == "metric not found" {
            http.Error(w, err.Error(), http.StatusNotFound)
        } else {
            http.Error(w, err.Error(), http.StatusBadRequest)
        }
        return
    }

    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(val))
}
