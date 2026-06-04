// Package buildinfo prints the build version, date and commit at startup.
//
// The three values are typically wired in from main packages whose globals
// are populated via -ldflags "-X main.buildVersion=… -X main.buildDate=…
// -X main.buildCommit=…". Empty strings are rendered as "N/A".
package buildinfo

import (
	"fmt"
	"io"
)

// Placeholder substituted into the output when a value is empty.
const Placeholder = "N/A"

// Print writes the formatted build info block to w.
func Print(w io.Writer, version, date, commit string) {
	fmt.Fprintf(w, "Build version: %s\n", orPlaceholder(version))
	fmt.Fprintf(w, "Build date: %s\n", orPlaceholder(date))
	fmt.Fprintf(w, "Build commit: %s\n", orPlaceholder(commit))
}

func orPlaceholder(s string) string {
	if s == "" {
		return Placeholder
	}
	return s
}
