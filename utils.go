package conncheck

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"strconv"
	"time"

	"golang.org/x/tools/go/analysis"
)

func newDiagnostic(arg ast.Expr, message string) *analysis.Diagnostic {
	return &analysis.Diagnostic{
		Pos:     arg.Pos(),
		End:     arg.End(),
		Message: message,
	}
}

func formatNode(node ast.Node) string {
	buf := new(bytes.Buffer)
	if err := format.Node(buf, token.NewFileSet(), node); err != nil {
		log.Printf("Error formatting expression: %v", err)
		return ""
	}

	return buf.String()
}

func printAST(node ast.Node) {
	fmt.Printf(">>>\n%s\n\n\n", formatNode(node))

	_ = ast.Fprint(os.Stdout, nil, node, nil)

	fmt.Println("--------------")
}

// basicLitToValue converts the BasicLit to an int64.
func basicLitToValue(arg *ast.BasicLit) (int64, bool) {
	intVal, err := strconv.ParseInt(arg.Value, 10, 64)
	if err != nil {
		return 0, false
	}

	return intVal, true
}

// binaryExprBasicLitToValue checks if X or Y is a BasicLit and returns the
// value.
func binaryExprBasicLitToValue(arg *ast.BinaryExpr) (int64, bool) {
	lit, litOk := arg.X.(*ast.BasicLit)
	if litOk {
		intVal, ok := basicLitToValue(lit)

		if ok {
			return intVal, true
		}
	}

	lit, litOk = arg.Y.(*ast.BasicLit)
	if litOk {
		intVal, ok := basicLitToValue(lit)

		if ok {
			return intVal, true
		}
	}

	return 0, false
}

// binaryExprCallExprToValue checks if X or Y of the BinaryExpr is a CallExpr
// and returns the value.
func binaryExprCallExprToValue(arg *ast.BinaryExpr) (int64, bool) {
	call, callOk := arg.X.(*ast.CallExpr)
	if callOk {
		intVal, ok := callExprToValue(call)
		if ok {
			return intVal, true
		}
	}

	call, callOk = arg.Y.(*ast.CallExpr)
	if callOk {
		intVal, ok := callExprToValue(call)
		if ok {
			return intVal, true
		}
	}

	return 0, false
}

// callExprToValue checks if the call expression is a call to time.Duration
func callExprToValue(call *ast.CallExpr) (int64, bool) {
	if len(call.Args) != 1 {
		return 0, false
	}

	if !isCallExprTimeDuration(call) {
		return 0, false
	}

	lit, litOk := call.Args[0].(*ast.BasicLit)
	if !litOk {
		return 0, false
	}

	intVal, ok := basicLitToValue(lit)
	if !ok {
		return 0, false
	}

	return intVal, true
}

// calcDuration calculates the duration based on the time unit and value.
func calcDuration(unit string, val int64) time.Duration {
	var t time.Duration

	switch unit {
	case "Nanosecond":
		t = time.Duration(val) * time.Nanosecond
	case "Microsecond":
		t = time.Duration(val) * time.Microsecond
	case "Millisecond":
		t = time.Duration(val) * time.Millisecond
	case "Second":
		t = time.Duration(val) * time.Second
	case "Minute":
		t = time.Duration(val) * time.Minute
	case "Hour":
		t = time.Duration(val) * time.Hour
	}

	return t
}

// isTimeGreaterThanMin checks if the time is greater than the minimum time.
func (cc *connCheck) isTimeGreaterThanMin(arg *ast.BinaryExpr) (bool, error) {
	// check for multiplication operator
	if arg.Op != token.MUL {
		return false, errNoCalc
	}

	hasUnit := false
	hasInt := false

	var intVal int64

	// check for time units
	unit, ok := cc.isTimeUnit(arg.X)
	if ok {
		hasUnit = true
	} else {
		unit, ok = cc.isTimeUnit(arg.Y)
		if ok {
			hasUnit = true
		}
	}

	if !hasUnit {
		return false, errNoCalc
	}

	// check for BasicLit
	intVal, hasInt = binaryExprBasicLitToValue(arg)

	// check for CallExpr
	if !hasInt {
		intVal, hasInt = binaryExprCallExprToValue(arg)
	}

	if !hasInt {
		return false, errNoCalc
	}

	t := calcDuration(unit, intVal)

	return t.Seconds() >= float64(cc.config.MinSeconds), nil
}

// isTimeUnit checks if the type is a time unit and is one of the valid time
// units.
func (cc *connCheck) isTimeUnit(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		selector, selectorOk := expr.(*ast.SelectorExpr)
		if !selectorOk || e.X == nil {
			return "", false
		}

		ident, identOk := e.X.(*ast.Ident)
		if !identOk {
			return "", false
		}

		// @TODO: switch to slices.Contains once Golangci-lint support Go 1.21
		if ident.Name == "time" && sliceContains(cc.config.ValidUnits.slice, selector.Sel.Name) {
			return selector.Sel.Name, true
		}
	default:
		return "", false
	}

	return "", false
}

// isCallTimeDuration checks if CallExpr is a call to time.Duration
func isCallExprTimeDuration(arg *ast.CallExpr) bool {
	selector, selectorOk := arg.Fun.(*ast.SelectorExpr)
	if !selectorOk || selector.X == nil {
		return false
	}

	return isSelectorExprTimeDuration(selector)
}

// isSelectorExprTimeDuration checks if SelectorExpr is of type time.Duration
func isSelectorExprTimeDuration(se *ast.SelectorExpr) bool {
	ident, identOk := se.X.(*ast.Ident)
	if !identOk {
		return false
	}

	return ident.Name == "time" && se.Sel.Name == "Duration"
}
