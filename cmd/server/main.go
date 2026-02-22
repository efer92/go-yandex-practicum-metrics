package main

import (
    "flag"
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/efer92/go-yandex-practicum-metrics/internal/handler"
    "github.com/efer92/go-yandex-practicum-metrics/internal/repository"
    "github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

var (
    flagAddr string
)

func parseFlags() {
    flag.StringVar(&flagAddr, "a", "localhost:8080", "HTTP server address")
    flag.Parse()
}

func main() {
    parseFlags()

    storage := repository.NewMemStorage()
    svc := service.NewMetricService(storage)
    h := handler.NewMetricHandler(svc)

    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Mount("/", h.Routes())

    log.Printf("Server starting on %s", flagAddr)
    if err := http.ListenAndServe(flagAddr, r); err != nil {
        log.Fatal(err)
    }
}
