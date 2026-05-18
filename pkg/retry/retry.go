// Package retry runs an operation with bounded retries on retriable errors.
package retry

import (
	"time"

	retrygo "github.com/avast/retry-go/v4"
)

var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do runs fn up to four times (one initial attempt and three retries) using the default delay
// schedule of 1s, 3s, 5s. A retry happens only when isRetriable returns true for the error.
func Do(fn func() error, isRetriable func(error) bool) error {
	return DoWithDelays(fn, isRetriable, delays)
}

// DoWithDelays runs fn with len(ivs)+1 total attempts, sleeping ivs[i] before retry i.
// The last interval is reused if more attempts are needed than there are entries.
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
