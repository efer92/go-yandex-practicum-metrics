package retry

import (
	"time"

	retrygo "github.com/avast/retry-go/v4"
)

var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do выполняет fn до 4 раз (1 основная + 3 повтора).
// Повторяет если isRetriable(err) == true.
func Do(fn func() error, isRetriable func(error) bool) error {
	return DoWithDelays(fn, isRetriable, delays)
}

func DoWithDelays(fn func() error, isRetriable func(error) bool, ivs []time.Duration) error {
	return retrygo.Do(
		fn,
		retrygo.Attempts(uint(len(ivs)+1)),
		retrygo.RetryIf(isRetriable),
		retrygo.LastErrorOnly(true),
		retrygo.DelayType(func(n uint, _ error, _ *retrygo.Config) time.Duration {
			if int(n) < len(ivs) {
				return ivs[n]
			}
			return ivs[len(ivs)-1]
		}),
	)
}
