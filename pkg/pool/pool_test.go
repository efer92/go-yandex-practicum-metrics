package pool

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type buffer struct {
	data    []int
	resetCt int
}

func (b *buffer) Reset() {
	b.resetCt++
	b.data = b.data[:0]
}

func TestPool_GetFromEmptyUsesFactory(t *testing.T) {
	p := New(func() *buffer { return &buffer{data: []int{1, 2, 3}} })

	v := p.Get()
	require.NotNil(t, v)
	assert.Equal(t, []int{1, 2, 3}, v.data)
}

func TestPool_PutResetsValueBeforeReturningToPool(t *testing.T) {
	p := New(func() *buffer { return &buffer{} })

	v := &buffer{data: []int{7, 8, 9}}
	p.Put(v)

	assert.Equal(t, 1, v.resetCt)
	assert.Empty(t, v.data)
}

func TestPool_FactoryCalledOncePerMissingValue(t *testing.T) {
	var calls int32
	p := New(func() *buffer {
		atomic.AddInt32(&calls, 1)
		return &buffer{}
	})

	a := p.Get()
	b := p.Get()
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
	assert.NotSame(t, a, b)

	p.Put(a)
	p.Put(b)
}

func TestPool_ConcurrentGetPutDoesNotRace(t *testing.T) {
	p := New(func() *buffer { return &buffer{} })

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := p.Get()
			v.data = append(v.data, 1, 2, 3)
			p.Put(v)
		}()
	}
	wg.Wait()
}
