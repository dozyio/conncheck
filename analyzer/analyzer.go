package analyzer

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"slices"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type connCheck struct{}

const (
	missingUnitErr = "include a time unit e.g. time.Second"
	operatorErr    = "operator is not -"
)

var (
	validUnits     = []string{"Second", "Minute", "Hour"}
	validTimeUnits = []string{"time.Second", "time.Minute", "time.Hour"}
)

func newConnCheck() *connCheck {
	return &connCheck{}
}

// New creates a new db-lifetime analyzer.
func New() *analysis.Analyzer {
	cc := newConnCheck()

	return &analysis.Analyzer{
		Name:     "conncheck",
		Doc:      "checks db.SetConnMaxLifetime is set to a reasonable value",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      cc.run,
	}
}

func (cc *connCheck) run(pass *analysis.Pass) (interface{}, error) {
	if !hasDbObj(pass) {
		return nil, nil
	}

	insp, inspOk := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !inspOk {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		&ast.CallExpr{},
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call, callOk := node.(*ast.CallExpr)
		if !callOk {
			return
		}

		selector, selectorOk := call.Fun.(*ast.SelectorExpr)
		if !selectorOk || selector.X == nil {
			return
		}

		ident, identOk := selector.X.(*ast.Ident)
		if !identOk {
			return
		}

		if ident.Name != "db" || selector.Sel.Name != "SetConnMaxLifetime" {
			return
		}

		checkArg(pass, call, call.Args[0])
	})

	return nil, nil
}

func checkArg(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr) {
	switch arg := arg.(type) {
	case *ast.BasicLit:
		if !isValidBasicLit(arg) {
			pass.Reportf(call.Pos(), "%s: `%s`", missingUnitErr, formatNode(call))
		}

	case *ast.UnaryExpr:
		if !isValidUnaryExpr(arg) {
			pass.Reportf(call.Pos(), "%s: `%s`", operatorErr, formatNode(call))
		}

	case *ast.BinaryExpr:
		if !isValidBinaryExpr(arg, pass) {
			pass.Reportf(call.Pos(), "%s: `%s`", missingUnitErr, formatNode(call))
		}

	case *ast.CallExpr:
		if !isValidCallExpr(arg) {
			pass.Reportf(call.Pos(), "%s: `%s`", missingUnitErr, formatNode(call))
		}

		// case *ast.SelectorExpr:
		// 	fmt.Printf("Found SelectorExpr: %+v\n", arg)
		//
		// case *ast.Ident:
		// 	fmt.Printf("Found Ident: %v\n", arg.Name)
		//
		// default:
		// 	fmt.Printf("Found an argument of an unknown kind: %T\n", arg)
	} //nolint:wsl // ignore
}

func hasDbObj(pass *analysis.Pass) bool {
	var dbObj types.Object

	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "database/sql" {
			dbObj = pkg.Scope().Lookup("DB")
			break
		}
	}

	return dbObj != nil
}

func isValidUnaryExpr(arg *ast.UnaryExpr) bool {
	return arg.Op == token.SUB
}

func isValidBasicLit(arg *ast.BasicLit) bool {
	if arg.Kind == token.INT {
		v, err := strconv.Atoi(arg.Value)
		if err != nil {
			return false
		}

		if v > 0 {
			return false
		}
	}

	return true
}

func isValidBinaryExpr(arg *ast.BinaryExpr, pass *analysis.Pass) bool {
	isUnit := false

	if arg.Op == token.MUL {
		if isTimeUnit(pass.TypesInfo.TypeOf(arg.X), arg.X) {
			isUnit = true
		}

		if isTimeUnit(pass.TypesInfo.TypeOf(arg.Y), arg.Y) {
			isUnit = true
		}
	}

	return isUnit
}

func isValidCallExpr(arg *ast.CallExpr) bool {
	selector, selectorOk := arg.Fun.(*ast.SelectorExpr)
	if !selectorOk || selector.X == nil {
		return true
	}

	ident, identOk := selector.X.(*ast.Ident)
	if !identOk {
		return true
	}

	if ident.Name == "time" && selector.Sel.Name == "Duration" {
		return false
	}

	return true
}

func formatNode(node ast.Node) string {
	buf := new(bytes.Buffer)

	err := format.Node(buf, token.NewFileSet(), node)
	if err != nil {
		log.Printf("error formatting expression: %v", err)
		return ""
	}

	return buf.String()
}

/* func printAST(node ast.Node) {
	fmt.Printf(">>>\n%s\n\n", formatNode(node))

	err := ast.Fprint(os.Stdout, nil, node, nil)
	if err != nil {
		fmt.Printf("error priting node: %v\n", err)
	}
} */

func isTimeUnit(x types.Type, expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		selector, selectorOk := expr.(*ast.SelectorExpr)
		if !selectorOk || e.X == nil {
			return false
		}

		ident, identOk := e.X.(*ast.Ident)
		if !identOk {
			return false
		}

		if ident.Name == "time" && slices.Contains(validUnits, selector.Sel.Name) {
			return true
		}
	default:
	}

	// Fallback to checking the type by string
	return slices.Contains(validTimeUnits, x.String())
}
