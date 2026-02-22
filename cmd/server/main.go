package main

import (
    "log"
    "net/http"
	"strings"

    "github.com/efer92/go-yandex-practicum-metrics/internal/handler"
    "github.com/efer92/go-yandex-practicum-metrics/internal/repository"
    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

func main() {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := handler.NewMetricHandler(svc)

    mux := http.NewServeMux()
    mux.HandleFunc("/update/", h.UpdateMetric)
    mux.HandleFunc("/value/", h.GetValue)

	// Проверяем URL до того, как передать в ServeMux
	// Обработка двойных слэшей в URL, иначе редирект
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Проверяем наличие двойных слешей
        if strings.Contains(r.URL.Path, "//") {
            http.Error(w, "Not found", http.StatusNotFound)
            return
        }
        mux.ServeHTTP(w, r)
    })
    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatal(err)
    }
}
