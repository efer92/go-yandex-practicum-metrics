package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type recordingObserver struct {
	events []Event
	err    error
}

func (o *recordingObserver) Receive(_ context.Context, e Event) error {
	o.events = append(o.events, e)
	return o.err
}

func TestPublisher_NotifyAllObservers(t *testing.T) {
	o1 := &recordingObserver{}
	o2 := &recordingObserver{}
	p := NewPublisher(zap.NewNop(), o1, o2)

	e := Event{TS: 100, Metrics: []string{"Alloc"}, IPAddress: "1.2.3.4"}
	p.Notify(context.Background(), e)

	assert.Equal(t, []Event{e}, o1.events)
	assert.Equal(t, []Event{e}, o2.events)
}

func TestPublisher_ObserverErrorDoesNotStop(t *testing.T) {
	failing := &recordingObserver{err: errors.New("boom")}
	ok := &recordingObserver{}
	p := NewPublisher(zap.NewNop(), failing, ok)

	e := Event{TS: 1, Metrics: []string{"X"}, IPAddress: "127.0.0.1"}
	p.Notify(context.Background(), e)

	assert.Equal(t, []Event{e}, failing.events)
	assert.Equal(t, []Event{e}, ok.events)
}
