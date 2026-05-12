// Package audit implements the Observer pattern for request auditing.
package audit

import (
	"context"

	"go.uber.org/zap"
)

// Event is a single audit record: a unix timestamp, the metric names handled, and the client IP.
type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Sink is the observer side of the audit pattern — a destination that receives Events.
type Sink interface {
	Receive(ctx context.Context, e Event) error
}

// Publisher fans an Event out to every registered Sink, logging per-sink errors.
type Publisher struct {
	sinks  []Sink
	logger *zap.Logger
}

// NewPublisher returns a Publisher wired to the given sinks.
func NewPublisher(logger *zap.Logger, sinks ...Sink) *Publisher {
	return &Publisher{sinks: sinks, logger: logger}
}

// Notify is nil-safe: callers can hold a *Publisher even when audit is disabled.
func (p *Publisher) Notify(ctx context.Context, e Event) {
	if p == nil {
		return
	}
	for _, s := range p.sinks {
		if err := s.Receive(ctx, e); err != nil && p.logger != nil {
			p.logger.Error("audit sink failed", zap.Error(err))
		}
	}
}
