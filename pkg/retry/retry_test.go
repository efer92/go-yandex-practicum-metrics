package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errRetriable = errors.New("retriable")
var errFatal = errors.New("fatal")

func isRetriable(err error) bool {
	return errors.Is(err, errRetriable)
}

// Нулевые интервалы для быстрых тестов
var zeroDelays = []time.Duration{0, 0, 0}

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := DoWithDelays(func() error {
		calls++
		return nil
	}, isRetriable, zeroDelays)

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessOnSecondAttempt(t *testing.T) {
	calls := 0
	err := DoWithDelays(func() error {
		calls++
		if calls < 2 {
			return errRetriable
		}
		return nil
	}, isRetriable, zeroDelays)

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestDo_ExhaustsAllRetries(t *testing.T) {
	calls := 0
	err := DoWithDelays(func() error {
		calls++
		return errRetriable
	}, isRetriable, zeroDelays)

	assert.ErrorIs(t, err, errRetriable)
	assert.Equal(t, 4, calls) // 1 основная + 3 повтора
}

func TestDo_FatalErrorNoRetry(t *testing.T) {
	calls := 0
	err := DoWithDelays(func() error {
		calls++
		return errFatal
	}, isRetriable, zeroDelays)

	assert.ErrorIs(t, err, errFatal)
	assert.Equal(t, 1, calls) // без повторов
}

func TestDo_RetriesWithIncreasingDelays(t *testing.T) {
	calls := 0
	ivs := []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}

	start := time.Now()
	DoWithDelays(func() error {
		calls++
		return errRetriable
	}, isRetriable, ivs)

	elapsed := time.Since(start)
	assert.Equal(t, 4, calls)
	assert.GreaterOrEqual(t, elapsed, 6*time.Millisecond) // 1+2+3
}
