package main

import (
	"log"
	"net/http"
	"os"

	"github.com/efer92/go-yandex-practicum-metrics/internal/handler"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := parseConfig(os.Args[1:])

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	log.Printf("Server starting on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		log.Fatal(err)
	}
}
