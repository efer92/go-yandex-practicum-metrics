package agent

import (
	"context"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"go.uber.org/zap"
)

type Agent struct {
	collector      *Collector
	sender         *Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	log            *zap.Logger
}

func NewWithConfig(serverURL string, pollInterval, reportInterval time.Duration, logger *zap.Logger, key string, rateLimit int) *Agent {
	if rateLimit <= 0 {
		rateLimit = 1
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

func (a *Agent) Run(ctx context.Context) {
	// Канал для передачи батчей метрик воркерам.
	jobs := make(chan []model.Metrics, a.rateLimit)

	// Запускаем worker pool — ограничивает число одновременных запросов.
	for i := 0; i < a.rateLimit; i++ {
		go a.worker(ctx, jobs)
	}

	// Горутина 1: сбор runtime-метрик.
	go a.runCollect(ctx)

	// Горутина 2: сбор extra-метрик (gopsutil).
	go a.runCollectExtra(ctx)

	// Горутина 3: отправка — кладёт снапшоты в канал jobs.
	a.runReport(ctx, jobs)
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
			a.log.Info("runtime metrics collected")
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
				a.log.Warn("extra metrics collection failed", zap.Error(err))
			} else {
				a.log.Info("extra metrics collected")
			}
		}
	}
}

// runReport периодически снимает снапшот и отправляет его в канал jobs.
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

// worker читает из канала jobs и отправляет батч на сервер.
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
				a.log.Info("metrics batch sent", zap.Int("count", len(metrics)))
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
