package main

import (
	"github.com/PiskarevSA/go-advanced/cmd/staticlint/noosexit"
	"github.com/PiskarevSA/go-advanced/cmd/staticlint/standard"
	"github.com/PiskarevSA/go-advanced/cmd/staticlint/staticcheck"
	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	analyzers := make([]*analysis.Analyzer, 0,
		len(standard.Analyzers)+len(staticcheck.Analyzers)+3)

	// стандартные статические анализаторы пакета golang.org/x/tools/go/analysis/passes
	analyzers = append(analyzers, standard.Analyzers...)

	// все анализаторы класса SA и по одному из остальных классов пакета staticcheck.io;
	analyzers = append(analyzers, staticcheck.Analyzers...)

	// два публичных анализатора
	analyzers = append(analyzers,
		errcheck.Analyzer,  // checking for unchecked errors in Go code
		bodyclose.Analyzer, // checks whether res.Body is correctly closed
	)

	// собственный анализатор, запрещающий использовать прямой вызов os.Exit в функции main пакета main
	analyzers = append(analyzers, noosexit.Analyzer)

	multichecker.Main(analyzers...)
}
