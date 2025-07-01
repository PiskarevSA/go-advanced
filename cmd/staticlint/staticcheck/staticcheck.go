package staticcheck

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
)

var Analyzers = analyzers()

func accept(name string) bool {
	// SA staticcheck
	if strings.HasPrefix(name, "SA") {
		return true
	}
	// S simple
	if name == "S1000" { // Use plain channel send or receive instead of single-case select
		return true
	}
	// ST stylecheck
	if name == "ST1001" { // Dot imports are discouraged
		return true
	}
	// QF quickfix
	if name == "QF1001" { // Apply De Morgan’s law
		return true
	}
	return false
}

func analyzers() []*analysis.Analyzer {
	var result []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if accept(v.Analyzer.Name) {
			result = append(result, v.Analyzer)
			return result
		}
	}
	return result
}
