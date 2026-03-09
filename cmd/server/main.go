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

func newRouter(svc *service.MetricService, logger *zap.Logger, pinger handler.Pinger) http.Handler {
	h := handler.NewMetricHandler(svc, logger, pinger)

	r := chi.NewRouter()
	r.Use(custommiddleware.Logger(logger))
	r.Use(custommiddleware.GzipMiddleware)
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	return r
}

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

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	g, ctx := errgroup.WithContext(sigCtx)

	var storage repository.Storage
	var fileSt *repository.FileBackedStorage
	var dbSt *repository.DBStorage

	if cfg.DatabaseDSN != "" {
		dbSt, err = repository.NewDBStorage(ctx, cfg.DatabaseDSN)
		if err != nil {
			logger.Warn("failed to connect to db, falling back to memory storage", zap.Error(err))
		} else {
			defer dbSt.Close()
			storage = dbSt
		}
	}

	if storage == nil && cfg.FileStoragePath != "" {
		fileSt = repository.NewFileBackedStorage(cfg.FileStoragePath, logger)
		if cfg.Restore {
			if err := fileSt.Load(); err != nil {
				logger.Warn("failed to load metrics from file", zap.Error(err))
			} else {
				logger.Info("metrics loaded from file", zap.String("path", cfg.FileStoragePath))
			}
		}
		storage = fileSt
	}

	if storage == nil {
		storage = repository.NewMemStorage()
	}

	svc := service.NewMetricService(storage)

	if fileSt != nil {
		switch {
		case cfg.StoreInterval == 0:
			svc.SetOnUpdate(fileSt.SaveOnUpdate)
		case cfg.StoreInterval > 0:
			fileSt.StartPeriodicSave(ctx, time.Duration(cfg.StoreInterval)*time.Second)
		}
	}

	var pinger handler.Pinger
	if dbSt != nil {
		pinger = dbSt
	}

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      newRouter(svc, logger, pinger),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("server starting",
		zap.String("addr", cfg.Addr),
		zap.String("database_dsn", cfg.DatabaseDSN),
		zap.Int("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	g.Go(func() error {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil {
		logger.Error("server stopped with error", zap.Error(err))
	}

	if fileSt != nil {
		if err := fileSt.Close(); err != nil {
			logger.Error("final save failed", zap.Error(err))
		}
	}
}
