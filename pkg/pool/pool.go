// Package pool provides a typed sync.Pool wrapper for objects whose state
// can be reset via a Reset() method.
//
// On Put the value is reset before being returned to the underlying pool,
// so callers can safely Get a clean instance later.
package pool

import "sync"

// Resetter is the constraint for Pool elements: any type whose method set
// contains Reset().
type Resetter interface {
	Reset()
}

// Pool stores values of type T for reuse. It is safe for concurrent use.
type Pool[T Resetter] struct {
	p sync.Pool
}

// New returns a Pool whose Get falls back to factory() when the pool is empty.
func New[T Resetter](factory func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any { return factory() },
		},
	}
}

// Get returns a value from the pool. When the pool is empty it allocates a
// fresh value via the factory passed to New.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put resets v and returns it to the pool for reuse.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
