package agent

import (
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"go.uber.org/zap"
)

type Agent struct {
	collector      *Collector
	sender         *Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	log            *zap.Logger
}

// NewWithConfig создает агента с кастомными настройками
func NewWithConfig(serverURL string, pollInterval, reportInterval time.Duration, logger *zap.Logger, key string) *Agent {
	return &Agent{
		collector:      NewCollector(),
		sender:         NewSender(serverURL, key),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		log:            logger,
	}
}

func (a *Agent) Run() {
	a.collector.Collect()
	a.log.Info("first metrics collected")

	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			a.collector.Collect()
			a.log.Info("metrics collected")

		case <-reportTicker.C:
			if err := a.report(); err != nil {
				a.log.Error("failed to report", zap.Error(err))
			} else {
				a.log.Info("metrics reported")
			}
		}
	}
}

func (a *Agent) report() error {
	snapshot := a.collector.GetSnapshot()
	metrics := convertToModelMetrics(snapshot)
	return a.sender.SendMetrics(metrics)
}

// convertToModelMetrics конвертирует внутренний снапшот для отправки
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
