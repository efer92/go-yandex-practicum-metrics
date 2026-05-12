// Package model defines the wire format used between the agent, server, and storage.
package model

// Metric type identifiers used in URLs and JSON payloads.
const (
	Counter = "counter"
	Gauge   = "gauge"
)

// Metrics is the flat wire representation of a single metric.
// Delta and Value are pointers so that a missing field is distinguishable from a zero value.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
