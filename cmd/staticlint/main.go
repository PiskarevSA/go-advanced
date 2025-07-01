package main

import (
	"github.com/PiskarevSA/go-advanced/cmd/staticlint/standard"
	"github.com/PiskarevSA/go-advanced/cmd/staticlint/staticcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	var analyzers []*analysis.Analyzer
	analyzers = append(analyzers, standard.Analyzers...)
	analyzers = append(analyzers, staticcheck.Analyzers...)
	multichecker.Main(analyzers...)
}
