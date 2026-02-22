package agent

import (
    "math/rand"
    "runtime"
)

// Metrics хранит собранные метрики
type Metrics struct {
    Gauge   map[string]float64
    Counter int64
}

// Collector собирает метрики из runtime
type Collector struct {
    metrics Metrics
}

func NewCollector() *Collector {
    return &Collector{
        metrics: Metrics{
            Gauge: make(map[string]float64),
        },
    }
}

// Collect собирает метрики из пакета runtime
func (c *Collector) Collect() {
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)

    // Все gauge метрики из runtime
    c.metrics.Gauge["Alloc"] = float64(memStats.Alloc)
    c.metrics.Gauge["BuckHashSys"] = float64(memStats.BuckHashSys)
    c.metrics.Gauge["Frees"] = float64(memStats.Frees)
    c.metrics.Gauge["GCCPUFraction"] = memStats.GCCPUFraction
    c.metrics.Gauge["GCSys"] = float64(memStats.GCSys)
    c.metrics.Gauge["HeapAlloc"] = float64(memStats.HeapAlloc)
    c.metrics.Gauge["HeapIdle"] = float64(memStats.HeapIdle)
    c.metrics.Gauge["HeapInuse"] = float64(memStats.HeapInuse)
    c.metrics.Gauge["HeapObjects"] = float64(memStats.HeapObjects)
    c.metrics.Gauge["HeapReleased"] = float64(memStats.HeapReleased)
    c.metrics.Gauge["HeapSys"] = float64(memStats.HeapSys)
    c.metrics.Gauge["LastGC"] = float64(memStats.LastGC)
    c.metrics.Gauge["Lookups"] = float64(memStats.Lookups)
    c.metrics.Gauge["MCacheInuse"] = float64(memStats.MCacheInuse)
    c.metrics.Gauge["MCacheSys"] = float64(memStats.MCacheSys)
    c.metrics.Gauge["MSpanInuse"] = float64(memStats.MSpanInuse)
    c.metrics.Gauge["MSpanSys"] = float64(memStats.MSpanSys)
    c.metrics.Gauge["Mallocs"] = float64(memStats.Mallocs)
    c.metrics.Gauge["NextGC"] = float64(memStats.NextGC)
    c.metrics.Gauge["NumForcedGC"] = float64(memStats.NumForcedGC)
    c.metrics.Gauge["NumGC"] = float64(memStats.NumGC)
    c.metrics.Gauge["OtherSys"] = float64(memStats.OtherSys)
    c.metrics.Gauge["PauseTotalNs"] = float64(memStats.PauseTotalNs)
    c.metrics.Gauge["StackInuse"] = float64(memStats.StackInuse)
    c.metrics.Gauge["StackSys"] = float64(memStats.StackSys)
    c.metrics.Gauge["Sys"] = float64(memStats.Sys)
    c.metrics.Gauge["TotalAlloc"] = float64(memStats.TotalAlloc)

    // Дополнительные метрики по заданию
    c.metrics.Gauge["RandomValue"] = rand.Float64()
    c.metrics.Counter++ // PollCount
}

// GetSnapshot возвращает копию текущих метрик и сбрасывает счетчик
func (c *Collector) GetSnapshot() Metrics {
    snapshot := c.metrics
    c.metrics.Counter = 0 // Сбрасываем после получения
    return snapshot
}
