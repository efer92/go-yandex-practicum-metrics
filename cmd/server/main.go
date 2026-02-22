package main

import (
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/efer92/go-yandex-practicum-metrics/internal/handler"
    "github.com/efer92/go-yandex-practicum-metrics/internal/repository"
    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

func main() {
    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := handler.NewMetricHandler(svc)

    r := chi.NewRouter()
    
    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Routes
    r.Mount("/", h.Routes())

    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", r); err != nil {
        log.Fatal(err)
    }
}
