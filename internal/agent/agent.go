package agent

import (
    "log"
    "time"
)

type Agent struct {
    collector      *Collector
    sender         *Sender
    pollInterval   time.Duration
    reportInterval time.Duration
}

// New создает агента с дефолтными настройками
func New() *Agent {
    return NewWithConfig(
        "http://localhost:8080",
        2*time.Second,
        10*time.Second,
    )
}

// NewWithConfig создает агента с кастомными настройками
func NewWithConfig(serverURL string, pollInterval, reportInterval time.Duration) *Agent {
    return &Agent{
        collector:      NewCollector(),
        sender:         NewSender(serverURL),
        pollInterval:   pollInterval,
        reportInterval: reportInterval,
    }
}

func (a *Agent) Run() {
    a.collector.Collect()
    log.Println("First metrics collected")

    pollTicker := time.NewTicker(a.pollInterval)
    reportTicker := time.NewTicker(a.reportInterval)
    defer pollTicker.Stop()
    defer reportTicker.Stop()

    for {
        select {
        case <-pollTicker.C:
            a.collector.Collect()
            log.Println("Metrics collected")

        case <-reportTicker.C:
            if err := a.report(); err != nil {
                log.Printf("Failed to report: %v", err)
            } else {
                log.Println("Metrics reported")
            }
        }
    }
}

func (a *Agent) report() error {
    metrics := a.collector.GetSnapshot()
    return a.sender.SendBatch(metrics)
}
