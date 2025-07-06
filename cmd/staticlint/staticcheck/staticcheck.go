package staticcheck

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
)

var Analyzers = analyzers()

func accept(name string) bool {
	// Все анализаторы SA staticcheck
	if strings.HasPrefix(name, "SA") {
		return true
	}
	// По одному из S simple, ST stylecheck и QF quickfix
	switch name {
	case "S1000", // Use plain channel send or receive instead of single-case select
		"ST1001", // Dot imports are discouraged
		"QF1001": // Apply De Morgan’s law
		return true
	}
	return false
}

func analyzers() []*analysis.Analyzer {
	var result []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if accept(v.Analyzer.Name) {
			result = append(result, v.Analyzer)
		}
	}
	return result
}
