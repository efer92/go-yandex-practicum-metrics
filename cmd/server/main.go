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
	"golang.org/x/sync/errgroup"
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

	// g.ctx отменяется когда любая горутина группы вернёт ошибку,
	// или когда получен SIGTERM/SIGINT через signal.NotifyContext.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	g, ctx := errgroup.WithContext(sigCtx)

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
			svc.SetOnUpdate(fileSt.SaveOnUpdate)
		case cfg.StoreInterval > 0:
			// Периодическое сохранение завершится когда ctx отменится
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

	// Горутина 1: HTTP-сервер.
	// Возвращает ошибку только если ListenAndServe упал не по Shutdown.
	g.Go(func() error {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Горутина 2: ждём отмены контекста (сигнал или ошибка сервера),
	// затем штатно останавливаем HTTP-сервер.
	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	// Ждём завершения обеих горутин
	if err := g.Wait(); err != nil {
		logger.Error("server stopped with error", zap.Error(err))
	}

	// Финальное сохранение — после остановки всех горутин
	if fileSt != nil {
		if err := fileSt.Close(); err != nil {
			logger.Error("final save failed", zap.Error(err))
		}
	}
}
