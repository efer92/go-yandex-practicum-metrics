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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Мапа всех gauge-метрик: имя -> значение
	gauges := map[string]float64{
		"Alloc":         float64(m.Alloc),
		"BuckHashSys":   float64(m.BuckHashSys),
		"Frees":         float64(m.Frees),
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         float64(m.GCSys),
		"HeapAlloc":     float64(m.HeapAlloc),
		"HeapIdle":      float64(m.HeapIdle),
		"HeapInuse":     float64(m.HeapInuse),
		"HeapObjects":   float64(m.HeapObjects),
		"HeapReleased":  float64(m.HeapReleased),
		"HeapSys":       float64(m.HeapSys),
		"LastGC":        float64(m.LastGC),
		"Lookups":       float64(m.Lookups),
		"MCacheInuse":   float64(m.MCacheInuse),
		"MCacheSys":     float64(m.MCacheSys),
		"MSpanInuse":    float64(m.MSpanInuse),
		"MSpanSys":      float64(m.MSpanSys),
		"Mallocs":       float64(m.Mallocs),
		"NextGC":        float64(m.NextGC),
		"NumForcedGC":   float64(m.NumForcedGC),
		"NumGC":         float64(m.NumGC),
		"OtherSys":      float64(m.OtherSys),
		"PauseTotalNs":  float64(m.PauseTotalNs),
		"StackInuse":    float64(m.StackInuse),
		"StackSys":      float64(m.StackSys),
		"Sys":           float64(m.Sys),
		"TotalAlloc":    float64(m.TotalAlloc),
		"RandomValue":   rand.Float64(),
	}

	// Копируем в хранилище метрик
	for name, value := range gauges {
		c.metrics.Gauge[name] = value
	}

	c.metrics.Counter++ // PollCount
}

// GetSnapshot возвращает копию текущих метрик и сбрасывает счетчик
func (c *Collector) GetSnapshot() Metrics {
	snapshot := c.metrics
	c.metrics.Counter = 0 // Сбрасываем после получения
	return snapshot
}
