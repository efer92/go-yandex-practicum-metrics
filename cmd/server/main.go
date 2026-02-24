package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/config"
	"github.com/efer92/go-yandex-practicum-metrics/internal/handler"
	custommiddleware "github.com/efer92/go-yandex-practicum-metrics/internal/middleware"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	var storage repository.Storage
	var fileSt *repository.FileBackedStorage

	if cfg.FileStoragePath != "" {
		fileSt = repository.NewFileBackedStorage(cfg.FileStoragePath, logger)
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

	if fileSt != nil {
		switch {
		case cfg.StoreInterval == 0:
			svc.SetOnUpdate(fileSt.SaveOnUpdate())
		case cfg.StoreInterval > 0:
			fileSt.StartPeriodicSave(ctx, time.Duration(cfg.StoreInterval)*time.Second)
		}
	}

	h := handler.NewMetricHandler(svc, logger)

	r := chi.NewRouter()
	r.Use(custommiddleware.Logger(logger))
	r.Use(custommiddleware.GzipMiddleware)
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	srv := &http.Server{Addr: cfg.Addr, Handler: r}

	logger.Info("server starting",
		zap.String("addr", cfg.Addr),
		zap.Int("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	// Запускаем HTTP-сервер в горутине
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			// ErrServerClosed — штатное завершение после Shutdown(), не ошибка
			serverErr <- err
		}
		close(serverErr)
	}()

	// Ждём сигнала завершения или ошибки сервера
	select {
	case <-ctx.Done():
		logger.Info("shutting down...")
	case err := <-serverErr:
		logger.Error("server error", zap.Error(err))
		stop()
	}

	// Останавливаем HTTP-сервер — ждём завершения активных соединений
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	// Ждём завершения горутины сервера
	for err := range serverErr {
		logger.Error("server error after shutdown", zap.Error(err))
	}

	// Финальное сохранение — после остановки всех горутин
	if fileSt != nil {
		if err := fileSt.Close(); err != nil {
			logger.Error("final save failed", zap.Error(err))
		}
	}
}
