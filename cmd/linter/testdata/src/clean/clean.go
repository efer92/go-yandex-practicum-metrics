package clean

import (
	"errors"
	"fmt"
)

// No forbidden patterns here.
func compute(x int) (int, error) {
	if x < 0 {
		return 0, errors.New("negative")
	}
	return x * 2, nil
}

func report(v int) string {
	return fmt.Sprintf("v=%d", v)
}
