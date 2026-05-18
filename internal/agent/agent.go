// Package agent collects runtime/system metrics and ships them to a metrics server.
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"go.uber.org/zap"
)

// Agent owns a Collector and a Sender and coordinates the poll/report loops.
type Agent struct {
	collector      *Collector
	sender         *Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	log            *zap.Logger
}

// NewWithConfig builds an Agent. The process is terminated with logger.Fatal when rateLimit <= 0.
func NewWithConfig(serverURL string, pollInterval, reportInterval time.Duration, logger *zap.Logger, key string, rateLimit int) *Agent {
	if rateLimit <= 0 {
		logger.Fatal("rateLimit must be > 0", zap.Int("rateLimit", rateLimit))
	}
	return &Agent{
		collector:      NewCollector(),
		sender:         NewSender(serverURL, key),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		log:            logger,
	}
}

// Run starts the agent loops and blocks until ctx is cancelled.
// All spawned goroutines are awaited before Run returns (graceful shutdown).
func (a *Agent) Run(ctx context.Context) {
	jobs := make(chan []model.Metrics, a.rateLimit)

	var wg sync.WaitGroup

	// Worker pool — ограничивает число одновременных запросов.
	for i := 0; i < a.rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.worker(ctx, jobs)
		}()
	}

	// Горутина 1: сбор runtime-метрик.
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runCollect(ctx)
	}()

	// Горутина 2: сбор extra-метрик.
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runCollectExtra(ctx)
	}()

	// Горутина 3: отправка снапшотов в канал jobs.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		a.runReport(ctx, jobs)
	}()

	wg.Wait()
	a.log.Info("agent stopped")
}

// runCollect периодически вызывает Collect.
func (a *Agent) runCollect(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	a.collector.Collect()
	a.log.Info("first metrics collected")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.collector.Collect()
			a.log.Debug("runtime metrics collected")
		}
	}
}

// runCollectExtra периодически вызывает CollectExtra.
func (a *Agent) runCollectExtra(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.collector.CollectExtra(); err != nil {
				a.log.Error("extra metrics collection failed", zap.Error(err))
			} else {
				a.log.Debug("extra metrics collected")
			}
		}
	}
}

// runReport периодически снимает снапшот и кладёт его в канал jobs.
// Закрывает jobs при выходе — воркеры узнают о завершении через закрытый канал.
func (a *Agent) runReport(ctx context.Context, jobs chan<- []model.Metrics) {
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := a.collector.GetSnapshot()
			metrics := convertToModelMetrics(snapshot)
			select {
			case jobs <- metrics:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (a *Agent) worker(ctx context.Context, jobs <-chan []model.Metrics) {
	for {
		select {
		case <-ctx.Done():
			return
		case metrics, ok := <-jobs:
			if !ok {
				return
			}
			if err := a.sender.SendBatch(metrics); err != nil {
				a.log.Error("failed to send batch", zap.Error(err))
			} else {
				a.log.Debug("metrics batch sent", zap.Int("count", len(metrics)))
			}
		}
	}
}

// convertToModelMetrics конвертирует снапшот в []model.Metrics.
func convertToModelMetrics(snapshot Metrics) []model.Metrics {
	result := make([]model.Metrics, 0, len(snapshot.Gauge)+1)

	for name, val := range snapshot.Gauge {
		v := val
		result = append(result, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	delta := snapshot.Counter
	result = append(result, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &delta,
	})

	return result
}
