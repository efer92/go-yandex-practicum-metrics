package panicpkg

import "errors"

func boom() error {
	return errors.New("boom")
}

func usePanic() {
	panic("nope") // want "panic call is forbidden"
}

func nested() {
	if err := boom(); err != nil {
		panic(err) // want "panic call is forbidden"
	}
}
