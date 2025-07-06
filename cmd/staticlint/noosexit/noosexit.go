package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "disallows direct call to os.Exit in main.main function (except first defer statement)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	// package should be main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// skip generated files
		if len(file.Comments) > 0 && strings.Contains(file.Comments[0].Text(), "generated") {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			// function should be main
			if !ok || fn.Name.Name != "main" || fn.Recv != nil {
				continue
			}

			firstDefer := true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				// skip defer statements, because os.Exit inside first defer is ok
				if _, ok := n.(*ast.DeferStmt); ok && firstDefer {
					firstDefer = false
					return false
				}

				// should be function call
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// should be os.Exit
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Exit" {
					return true
				}

				xIdent, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}

				obj := pass.TypesInfo.Uses[xIdent]
				if obj == nil {
					return true
				}

				pkgName, ok := obj.(*types.PkgName)
				if !ok || pkgName.Imported().Path() != "os" {
					return true
				}

				pass.Reportf(call.Lparen, "direct call to os.Exit is not allowed in main.main (except first defer statement)")
				return true
			})
		}
	}

	return nil, nil
}
