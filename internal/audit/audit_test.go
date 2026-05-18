package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type recordingSink struct {
	events []Event
	err    error
}

func (s *recordingSink) Receive(_ context.Context, e Event) error {
	s.events = append(s.events, e)
	return s.err
}

func TestPublisher_NotifyAllSinks(t *testing.T) {
	s1 := &recordingSink{}
	s2 := &recordingSink{}
	p := NewPublisher(zap.NewNop(), s1, s2)

	e := Event{TS: 100, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}
	p.Notify(context.Background(), e)

	assert.Equal(t, []Event{e}, s1.events)
	assert.Equal(t, []Event{e}, s2.events)
}

func TestPublisher_NilSafe(t *testing.T) {
	var p *Publisher
	assert.NotPanics(t, func() {
		p.Notify(context.Background(), Event{})
	})
}

func TestPublisher_SinkErrorDoesNotStop(t *testing.T) {
	failing := &recordingSink{err: errors.New("boom")}
	ok := &recordingSink{}
	p := NewPublisher(zap.NewNop(), failing, ok)

	e := Event{TS: 1, Metrics: []string{"X"}, IPAddress: "127.0.0.1"}
	p.Notify(context.Background(), e)

	assert.Equal(t, []Event{e}, failing.events)
	assert.Equal(t, []Event{e}, ok.events)
}
