// Package for static analyze code
package main

import (
	"go/ast"
	"strings"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// Analyze "SA..." prefix of staticcheck analyzers
const saStaticCheckPrefix = "SA"

// Analyzers lists
var (
	allAnalyzers, basicAnalizers, saStaticCheckAnalizers, customStaticCheckAnalizers []*analysis.Analyzer
	allCustomAnalyzers []*lint.Analyzer
	
	customNamesStaticCheckAnalyzers = map[string]bool {
		"S1000": true,
		"S1001": true,
		"S1005": true,
		"S1011": true,
		"S1012": true,

		"ST1000": true,
		"ST1003": true,
		"ST1005": true,

		"QF1002": true,
		"QF1003": true,
		"QF1004": true,
		"QF1012": true,
	}

	// osExit analyzer
	osExitAstAnalyze = &analysis.Analyzer{
		Name: "osexitcheck",
		Doc: "Check os.Exit in main",
		Run: osExitChecker,
	}
)

// osExitChecker function try to found os.Exit in main function
func osExitChecker(pass *analysis.Pass) (interface{}, error) {
	mainFound := false
	for _, v := range pass.Files {
		if v.Name.Name == "main" {
			ast.Inspect(v, func(n ast.Node) bool {
				if f, ok := n.(*ast.FuncDecl); ok && f.Name.Name == "main" {
					mainFound = true
				}
				if s, ok := n.(*ast.SelectorExpr); ok && mainFound && s.Sel.Name == "Exit" {
					if i, ok := s.X.(*ast.Ident); ok && i.Name == "os" {
						mainFound = false
						pass.Reportf(i.NamePos, "direct os.Exit found in main/main")
					}
				}
				return true
			})
			return nil, nil
		}
	}
	return nil, nil
}

// Main func for process analyzers
func main() {
	// Basic analyzers aka "go vet..."
	basicAnalizers = []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		pkgfact.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
	}

	// Add specific "^SA" analizers from staticcheck package
	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, saStaticCheckPrefix) {
			saStaticCheckAnalizers = append(saStaticCheckAnalizers, v.Analyzer)
		}
	}

	// Add custom staticcheck analyzers
	allCustomAnalyzers = append(allCustomAnalyzers, simple.Analyzers...)
	allCustomAnalyzers = append(allCustomAnalyzers, stylecheck.Analyzers...)
	allCustomAnalyzers = append(allCustomAnalyzers, quickfix.Analyzers...)
	for _, v := range allCustomAnalyzers {
		if customNamesStaticCheckAnalyzers[v.Analyzer.Name] {
			customStaticCheckAnalizers = append(customStaticCheckAnalizers, v.Analyzer)
		}
	}

	// Add public analizers (errcheck - github.com/kisielk/errcheck and github.com/gordonklaus/ineffassign/pkg/ineffassign)
	customStaticCheckAnalizers = append(customStaticCheckAnalizers, errcheck.Analyzer, ineffassign.Analyzer)

	// Add os.Exit analyzer
	customStaticCheckAnalizers = append(customStaticCheckAnalizers, osExitAstAnalyze)

	// Union all analyzers
	allAnalyzers = append(allAnalyzers, basicAnalizers...)
	allAnalyzers = append(allAnalyzers, saStaticCheckAnalizers...)
	allAnalyzers = append(allAnalyzers, customStaticCheckAnalizers...)

	// Multichecker analyzers run
	multichecker.Main(allAnalyzers...)
}