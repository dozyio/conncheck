package conncheck_test

import (
	"testing"

	"github.com/dozyio/conncheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), conncheck.New(), "p")
}
