package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Используем FileBackedStorage если задан путь к файлу
	var storage repository.Storage
	var fileSt *repository.FileBackedStorage

	if cfg.FileStoragePath != "" {
		fileSt = repository.NewFileBackedStorage(cfg.FileStoragePath)

		if cfg.Restore {
			if err := fileSt.Load(); err != nil {
				logger.Warn("failed to load metrics from file", zap.Error(err))
			} else {
				logger.Info("metrics loaded from file", zap.String("path", cfg.FileStoragePath))
			}
		}
		storage = fileSt
	} else {
		storage = repository.NewMemStorage()
	}

	svc := service.NewMetricService(storage)

	// Синхронная запись при каждом обновлении (StoreInterval == 0)
	if fileSt != nil && cfg.StoreInterval == 0 {
		svc.SetOnUpdate(func() {
			if err := fileSt.Save(); err != nil {
				logger.Error("sync save failed", zap.Error(err))
			}
		})
	}

	h := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Use(custommiddleware.Logger(logger))
	r.Use(custommiddleware.GzipMiddleware)
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	// Периодическое сохранение (StoreInterval > 0)
	if fileSt != nil && cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := fileSt.Save(); err != nil {
					logger.Error("periodic save failed", zap.Error(err))
				} else {
					logger.Info("metrics saved", zap.String("path", cfg.FileStoragePath))
				}
			}
		}()
	}

	// Сохранение при завершении
	if fileSt != nil {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-quit
			logger.Info("shutting down, saving metrics...")
			if err := fileSt.Save(); err != nil {
				logger.Error("final save failed", zap.Error(err))
			}
			os.Exit(0)
		}()
	}

	logger.Info("server starting",
		zap.String("addr", cfg.Addr),
		zap.Int("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
