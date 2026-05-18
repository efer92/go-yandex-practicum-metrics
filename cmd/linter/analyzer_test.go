package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoExit_PanicDetected(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noExitAnalyzer, "panicpkg")
}

func TestNoExit_FatalAndExitOutsideMain(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noExitAnalyzer, "fataloutside")
}

func TestNoExit_AllowedInsideMainMain(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noExitAnalyzer, "mainok")
}

func TestNoExit_CleanFile(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noExitAnalyzer, "clean")
}
