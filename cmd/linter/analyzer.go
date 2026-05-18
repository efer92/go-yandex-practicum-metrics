package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// noExitAnalyzer flags three forbidden patterns:
//   - the built-in panic call (anywhere)
//   - log.Fatal / log.Fatalf / log.Fatalln (outside main() of package main)
//   - os.Exit (outside main() of package main)
//
// _test.go files and generated files are skipped.
var noExitAnalyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "reports panic, log.Fatal*, and os.Exit calls; the latter two are allowed only inside main() of package main",
	Run:  runNoExit,
}

func runNoExit(pass *analysis.Pass) (interface{}, error) {
	isMainPkg := pass.Pkg.Name() == "main"

	for _, file := range pass.Files {
		if isGenerated(file) {
			continue
		}
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			inMainFunc := isMainPkg && fn.Recv == nil && fn.Name != nil && fn.Name.Name == "main"
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				checkCall(pass, call, inMainFunc)
				return true
			})
		}
	}
	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, inMainFunc bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if fun.Name == "panic" {
			pass.Reportf(call.Pos(), "panic call is forbidden")
		}
	case *ast.SelectorExpr:
		pkgID, ok := fun.X.(*ast.Ident)
		if !ok {
			return
		}
		switch {
		case pkgID.Name == "log" && strings.HasPrefix(fun.Sel.Name, "Fatal"):
			if !inMainFunc {
				pass.Reportf(call.Pos(), "log.%s is forbidden outside main() of package main", fun.Sel.Name)
			}
		case pkgID.Name == "os" && fun.Sel.Name == "Exit":
			if !inMainFunc {
				pass.Reportf(call.Pos(), "os.Exit is forbidden outside main() of package main")
			}
		}
	}
}

func isGenerated(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}
