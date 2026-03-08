package retry

import (
	"time"
)

var intervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do выполняет fn до 4 раз (1 основная + 3 повтора).
// Повторяет если isRetriable(err) == true.
func Do(fn func() error, isRetriable func(error) bool) error {
	err := fn()
	if err == nil {
		return nil
	}

	for _, interval := range intervals {
		if !isRetriable(err) {
			return err
		}
		time.Sleep(interval)
		err = fn()
		if err == nil {
			return nil
		}
	}

	return err
}
