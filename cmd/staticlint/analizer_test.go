package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
	"honnef.co/go/tools/stylecheck"
)

func TestMyAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), osExitAstAnalyze, ".")
}

func TestGetStaticCheckAnalyzers(t *testing.T) {
	analyzers := getStaticCheckAnalyzers("SA")
	require.Greater(t, len(analyzers), 0)
}

func TestGetCustomHonnefAnalyzers(t *testing.T) {
	analyzers := getCustomHonnefAnalyzers(stylecheck.Analyzers, customNamesHonnefAnalyzers)
	require.Greater(t, len(analyzers), 0)
}
