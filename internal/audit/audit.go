// Package audit implements the Observer pattern for request auditing.
package audit

import (
	"context"
	"io"

	"go.uber.org/zap"
)

// Event is a single audit record: a unix timestamp, the metric names handled, and the client IP.
type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer is the observer side of the audit pattern — a destination that receives Events.
type Observer interface {
	Receive(ctx context.Context, e Event) error
}

// Publisher fans an Event out to every registered Observer, logging per-observer errors.
type Publisher struct {
	observers []Observer
	logger    *zap.Logger
}

// NewPublisher returns a Publisher wired to the given observers.
func NewPublisher(logger *zap.Logger, observers ...Observer) *Publisher {
	return &Publisher{observers: observers, logger: logger}
}

// Notify delivers e to every observer sequentially.
// Callers that must not block on slow observers should invoke Notify from a goroutine.
func (p *Publisher) Notify(ctx context.Context, e Event) {
	for _, o := range p.observers {
		if err := o.Receive(ctx, e); err != nil && p.logger != nil {
			p.logger.Error("audit observer failed", zap.Error(err))
		}
	}
}

// Close releases observers that implement io.Closer.
func (p *Publisher) Close() error {
	var firstErr error
	for _, o := range p.observers {
		if c, ok := o.(io.Closer); ok {
			if err := c.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
