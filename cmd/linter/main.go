// Linter is a singlechecker entry point for the project's custom static analyzer.
//
// Usage:
//
//	go run ./cmd/linter ./...
//
// It reports any panic call and any log.Fatal* / os.Exit call outside main()
// of package main. Test files (_test.go) and generated files are skipped.
package main

import "golang.org/x/tools/go/analysis/singlechecker"

func main() {
	singlechecker.Main(noExitAnalyzer)
}
