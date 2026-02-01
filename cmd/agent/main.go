package main

import (
    "github.com/efer92/go-yandex-practicum-metrics/internal/agent"
)

func main() {
    ag := agent.New()
    ag.Run()
}
