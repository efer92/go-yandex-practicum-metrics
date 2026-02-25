package main

import (
	"log"
	"net/http"
	"os"

	"github.com/efer92/go-yandex-practicum-metrics/internal/handler"
	custommiddleware "github.com/efer92/go-yandex-practicum-metrics/internal/middleware"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	cfg := parseConfig(os.Args[1:])

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Use(custommiddleware.Logger(logger)) // наш zap-логгер
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	logger.Info("server starting", zap.String("addr", cfg.Addr))
	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
