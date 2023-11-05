package main

import (
	analyzer "github.com/dozyio/conncheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.New())
}
