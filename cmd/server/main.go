package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/audit"
	"github.com/efer92/go-yandex-practicum-metrics/internal/config"
	"github.com/efer92/go-yandex-practicum-metrics/internal/handler"
	custommiddleware "github.com/efer92/go-yandex-practicum-metrics/internal/middleware"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/buildinfo"
	"github.com/efer92/go-yandex-practicum-metrics/pkg/crypto"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func newRouter(svc *service.MetricService, logger *zap.Logger, pinger handler.Pinger, key string, publisher *audit.Publisher, privKey *rsa.PrivateKey, trustedSubnet func(http.Handler) http.Handler) http.Handler {
	h := handler.NewMetricHandler(svc, logger, pinger, publisher)

	r := chi.NewRouter()
	r.Use(custommiddleware.Logger(logger))
	// trustedSubnet first so we reject untrusted callers before doing any
	// crypto or hashing work on a body we are going to throw away anyway.
	if trustedSubnet != nil {
		r.Use(trustedSubnet)
	}
	if m := custommiddleware.CryptoMiddleware(privKey, logger); m != nil {
		r.Use(m)
	}
	r.Use(custommiddleware.GzipMiddleware)
	if m := custommiddleware.HashMiddleware(key, logger); m != nil {
		r.Use(m)
	}
	r.Use(middleware.Recoverer)
	r.Mount("/", h.Routes())

	return r
}

// buildAuditPublisher returns nil when no audit observer is configured.
func buildAuditPublisher(cfg config.ServerConfig, logger *zap.Logger) (*audit.Publisher, error) {
	var observers []audit.Observer
	if cfg.AuditFile != "" {
		obs, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, err
		}
		observers = append(observers, obs)
	}
	if cfg.AuditURL != "" {
		observers = append(observers, audit.NewHTTPObserver(cfg.AuditURL))
	}
	if len(observers) == 0 {
		return nil, nil
	}
	return audit.NewPublisher(logger, observers...), nil
}

// Populated via -ldflags "-X main.buildVersion=… -X main.buildDate=… -X main.buildCommit=…".
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	cfg, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
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

	publisher, err := buildAuditPublisher(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init audit publisher", zap.Error(err))
	}
	if publisher != nil {
		defer func() {
			if err := publisher.Close(); err != nil {
				logger.Error("audit publisher close failed", zap.Error(err))
			}
		}()
	}

	var privKey *rsa.PrivateKey
	if cfg.CryptoKey != "" {
		privKey, err = crypto.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			log.Fatalf("load crypto private key: %v", err)
		}
	}

	trustedSubnetMw, err := custommiddleware.TrustedSubnetMiddleware(cfg.TrustedSubnet, logger)
	if err != nil {
		log.Fatalf("trusted subnet: %v", err)
	}

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      newRouter(svc, logger, pinger, cfg.Key, publisher, privKey, trustedSubnetMw),
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
		zap.String("audit_file", cfg.AuditFile),
		zap.String("audit_url", cfg.AuditURL),
		zap.String("trusted_subnet", cfg.TrustedSubnet),
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
