package agent

import (
    "log"
    "time"
)

const (
    pollInterval   = 2 * time.Second
    reportInterval = 10 * time.Second
    serverURL      = "http://localhost:8080"
)

// Agent координирует сбор и отправку метрик
type Agent struct {
    collector *Collector
    sender    *Sender
}

func New() *Agent {
    return &Agent{
        collector: NewCollector(),
        sender:    NewSender(serverURL),
    }
}

// Run запускает агента (блокирующий вызов)
func (a *Agent) Run() {
    // Первый сбор сразу
    a.collector.Collect()
    log.Println("Agent started")

    pollTicker := time.NewTicker(pollInterval)
    reportTicker := time.NewTicker(reportInterval)
    defer pollTicker.Stop()
    defer reportTicker.Stop()

    for {
        select {
        case <-pollTicker.C:
            a.collector.Collect()
            log.Println("Metrics collected")

        case <-reportTicker.C:
            if err := a.report(); err != nil {
                log.Printf("Failed to report metrics: %v", err)
            } else {
                log.Println("Metrics reported successfully")
            }
        }
    }
}

func (a *Agent) report() error {
    metrics := a.collector.GetSnapshot()
    return a.sender.SendBatch(metrics)
}