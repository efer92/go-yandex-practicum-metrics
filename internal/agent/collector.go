package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Metrics is a single snapshot of agent-side metric state.
type Metrics struct {
	Gauge   map[string]float64
	Counter int64
}

// Collector accumulates gauge values from runtime/gopsutil and a counter of poll ticks.
type Collector struct {
	mu      sync.Mutex
	metrics Metrics
}

// NewCollector returns an empty Collector.
func NewCollector() *Collector {
	return &Collector{
		metrics: Metrics{Gauge: make(map[string]float64)},
	}
}

// Collect samples runtime.MemStats, updates the gauges, and increments the poll counter.
func (c *Collector) Collect() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.Gauge["Alloc"] = float64(ms.Alloc)
	c.metrics.Gauge["BuckHashSys"] = float64(ms.BuckHashSys)
	c.metrics.Gauge["Frees"] = float64(ms.Frees)
	c.metrics.Gauge["GCCPUFraction"] = ms.GCCPUFraction
	c.metrics.Gauge["GCSys"] = float64(ms.GCSys)
	c.metrics.Gauge["HeapAlloc"] = float64(ms.HeapAlloc)
	c.metrics.Gauge["HeapIdle"] = float64(ms.HeapIdle)
	c.metrics.Gauge["HeapInuse"] = float64(ms.HeapInuse)
	c.metrics.Gauge["HeapObjects"] = float64(ms.HeapObjects)
	c.metrics.Gauge["HeapReleased"] = float64(ms.HeapReleased)
	c.metrics.Gauge["HeapSys"] = float64(ms.HeapSys)
	c.metrics.Gauge["LastGC"] = float64(ms.LastGC)
	c.metrics.Gauge["Lookups"] = float64(ms.Lookups)
	c.metrics.Gauge["MCacheInuse"] = float64(ms.MCacheInuse)
	c.metrics.Gauge["MCacheSys"] = float64(ms.MCacheSys)
	c.metrics.Gauge["MSpanInuse"] = float64(ms.MSpanInuse)
	c.metrics.Gauge["MSpanSys"] = float64(ms.MSpanSys)
	c.metrics.Gauge["Mallocs"] = float64(ms.Mallocs)
	c.metrics.Gauge["NextGC"] = float64(ms.NextGC)
	c.metrics.Gauge["NumForcedGC"] = float64(ms.NumForcedGC)
	c.metrics.Gauge["NumGC"] = float64(ms.NumGC)
	c.metrics.Gauge["OtherSys"] = float64(ms.OtherSys)
	c.metrics.Gauge["PauseTotalNs"] = float64(ms.PauseTotalNs)
	c.metrics.Gauge["StackInuse"] = float64(ms.StackInuse)
	c.metrics.Gauge["StackSys"] = float64(ms.StackSys)
	c.metrics.Gauge["Sys"] = float64(ms.Sys)
	c.metrics.Gauge["TotalAlloc"] = float64(ms.TotalAlloc)
	c.metrics.Gauge["RandomValue"] = rand.Float64()
	c.metrics.Counter++
}

// CollectExtra samples gopsutil-based memory and CPU metrics.
func (c *Collector) CollectExtra() error {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("get virtual memory: %w", err)
	}

	cpuPercents, err := cpu.Percent(0, true)
	if err != nil {
		return fmt.Errorf("get cpu percent: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.Gauge["TotalMemory"] = float64(vmStat.Total)
	c.metrics.Gauge["FreeMemory"] = float64(vmStat.Free)
	for i, pct := range cpuPercents {
		c.metrics.Gauge[fmt.Sprintf("CPUutilization%d", i+1)] = pct
	}

	return nil
}

// GetSnapshot returns a deep copy of the current metric state.
func (c *Collector) GetSnapshot() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := Metrics{
		Gauge:   make(map[string]float64, len(c.metrics.Gauge)),
		Counter: c.metrics.Counter,
	}
	for k, v := range c.metrics.Gauge {
		snapshot.Gauge[k] = v
	}
	return snapshot
}

// AckCounter subtracts delta from the poll counter after a successful upload,
// preserving any increments that arrived between snapshot and ack.
func (c *Collector) AckCounter(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.Counter -= delta
}
