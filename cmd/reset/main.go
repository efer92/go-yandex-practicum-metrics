// Command reset generates Reset() methods for structs marked with
// // generate:reset across the project.
//
// Usage:
//
//	go run ./cmd/reset            # walks current directory
//	go run ./cmd/reset ./internal # walks the given subtree
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/efer92/go-yandex-practicum-metrics/cmd/reset/generate"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := generate.Run(root); err != nil {
		log.Fatalf("reset: %v", err)
	}
	fmt.Println("reset: done")
}
