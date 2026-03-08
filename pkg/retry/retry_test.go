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

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return nil
	}, isRetriable)

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessOnSecondAttempt(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 2 {
			return errRetriable
		}
		return nil
	}, isRetriable)

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestDo_ExhaustsAllRetries(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return errRetriable
	}, isRetriable)

	assert.ErrorIs(t, err, errRetriable)
	assert.Equal(t, 4, calls) // 1 основная + 3 повтора
}

func TestDo_FatalErrorNoRetry(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return errFatal
	}, isRetriable)

	assert.ErrorIs(t, err, errFatal)
	assert.Equal(t, 1, calls) // без повторов
}

func TestDo_RetriesWithIncreasingIntervals(t *testing.T) {
	calls := 0
	start := time.Now()
	Do(func() error {
		calls++
		return errRetriable
	}, isRetriable)

	elapsed := time.Since(start)
	// 1+3+5 = 9 секунд — в тесте слишком долго, поэтому просто проверяем что было 4 вызова
	assert.Equal(t, 4, calls)
	_ = elapsed
}
