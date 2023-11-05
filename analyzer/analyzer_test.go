package analyzer_test

import (
	"testing"

	"github.com/dozyio/conncheck/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzer.New(), "p")
}
