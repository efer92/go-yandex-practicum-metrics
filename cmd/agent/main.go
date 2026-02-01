package main

import (
    "flag"
    "log"
    "time"

    "github.com/efer92/go-yandex-practicum-metrics/internal/agent"
)

var (
    flagAddr           string
    flagReportInterval int
    flagPollInterval   int
)

func parseFlags() {
    flag.StringVar(&flagAddr, "a", "localhost:8080", "HTTP server address")
    flag.IntVar(&flagReportInterval, "r", 10, "Report interval in seconds")
    flag.IntVar(&flagPollInterval, "p", 2, "Poll interval in seconds")
    flag.Parse()
}

func main() {
    parseFlags()

    // Формируем URL сервера
    serverURL := "http://" + flagAddr

    ag := agent.NewWithConfig(
        serverURL,
        time.Duration(flagPollInterval)*time.Second,
        time.Duration(flagReportInterval)*time.Second,
    )

    log.Printf("Agent started (server: %s, poll: %ds, report: %ds)",
        flagAddr, flagPollInterval, flagReportInterval)
    ag.Run()
}
