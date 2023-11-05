package conncheck

import (
	"errors"
	"flag"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const minSecondsDefault = 60

var (
	pkgsDefault       = []string{"database/sql", "gorm.io/gorm"}
	validUnitsDefault = []string{"Second", "Minute", "Hour"}
	printASTDefault   = false

	errMissingUnit          = errors.New("missing a valid time unit")
	errOperator             = errors.New("operator is not -")
	errPotentialMissingUnit = errors.New("potentially missing a time unit")
	errCalcLessThanMin      = errors.New("time is less than minimum required")
	errNoCalc               = errors.New("can't calculate time")
	errInvalidParam         = errors.New("invalid parameter")
)

type Config struct {
	pkgs       stringSliceValue
	validUnits stringSliceValue
	minSeconds uint64
	printAST   bool
}

type connCheck struct {
	config *Config
}

func newConncheck(config *Config) *connCheck {
	return &connCheck{
		config: config,
	}
}

func (cc *connCheck) flags() flag.FlagSet {
	flags := flag.NewFlagSet("", flag.ExitOnError)

	// If we have a config, we're not running from cli so don't use any flags
	if cc.config != nil {
		return *flags
	}

	cc.config = defaultConfig()

	flags.Var(&cc.config.pkgs, "packages", "A comma-separated list of packages to check against")
	flags.Var(&cc.config.validUnits, "timeunits", "A comma-separated list of time units to validate against")
	flags.Uint64Var(&cc.config.minSeconds, "minsec", minSecondsDefault, "The minimum seconds of SetConnMaxLifetime")
	flags.BoolVar(&cc.config.printAST, "printast", printASTDefault, "Print AST")

	return *flags
}

// New creates a new conncheck analyzer.
func New(config *Config) *analysis.Analyzer {
	cc := newConncheck(config)

	return &analysis.Analyzer{
		Name:     "conncheck",
		Doc:      "checks db.SetConnMaxLifetime is set to a reasonable value",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      cc.run,
		Flags:    cc.flags(),
	}
}

func defaultConfig() *Config {
	return &Config{
		pkgs: stringSliceValue{
			slice: pkgsDefault,
		},
		validUnits: stringSliceValue{
			slice: validUnitsDefault,
		},
		minSeconds: minSecondsDefault,
	}
}

func (cc *connCheck) run(pass *analysis.Pass) (interface{}, error) {
	if !cc.hasDbObj(pass) {
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
		if selector.Sel.Name != "SetConnMaxLifetime" {
			return
		}

		ident, identOk := selector.X.(*ast.Ident)
		if !identOk {
			return
		}

		if pass.TypesInfo.TypeOf(ident).String() != "*database/sql.DB" {
			return
		}

		cc.process(pass, call, node)
	})

	return nil, nil
}

// process checks if the call expression for SetConnMaxLifetime has a time
// duration.
func (cc *connCheck) process(pass *analysis.Pass, call *ast.CallExpr, node ast.Node) {
	if cc.config.printAST {
		printAST(node)
	}

	arg := call.Args[0]

	switch arg := arg.(type) {
	case *ast.BasicLit:
		if !cc.isValidBasicLit(arg) {
			pass.Report(*newDiagnostic(arg, errMissingUnit.Error()))
		}

	case *ast.UnaryExpr:
		if !cc.isValidUnaryExpr(arg) {
			pass.Report(*newDiagnostic(arg, errOperator.Error()))
		}

	case *ast.BinaryExpr:
		if !cc.isValidBinaryExpr(arg) {
			pass.Report(*newDiagnostic(arg, errMissingUnit.Error()))
		}

		res, err := cc.isTimeGreaterThanMin(arg)
		if err == nil {
			if !res {
				pass.Report(*newDiagnostic(arg, errCalcLessThanMin.Error()))
			}
		}

	case *ast.CallExpr:
		err := cc.isValidCallExpr(arg)
		if err != nil {
			pass.Report(*newDiagnostic(arg, err.Error()))
		}

	case *ast.SelectorExpr:
		err := cc.isValidSelectorExpr(arg)
		if err != nil {
			pass.Report(*newDiagnostic(arg, err.Error()))
		}

	case *ast.Ident:
		err := cc.isValidIdent(arg)
		if err != nil {
			pass.Report(*newDiagnostic(arg, err.Error()))
		}

		// default:
		// 	fmt.Printf("Found an argument of an unknown kind: %T\n", arg)
	} //nolint:wsl // ignore
}

// hasDbObj checks if the package uses one of the packages listed in pkgs
func (cc *connCheck) hasDbObj(pass *analysis.Pass) bool {
	var dbObj types.Object

	for _, pkg := range pass.Pkg.Imports() {
		if slices.Contains(cc.config.pkgs.slice, pkg.Path()) {
			dbObj = pkg.Scope().Lookup("DB")

			if dbObj != nil {
				return true
			}
		}
	}

	return false
}

func (cc *connCheck) isValidIdent(ident *ast.Ident) error {
	if ident.Obj.Kind != ast.Var {
		return nil
	}

	if ident.Obj.Decl == nil {
		return nil
	}

	decl, declOk := ident.Obj.Decl.(*ast.AssignStmt)
	if !declOk {
		return nil
	}

	if len(decl.Rhs) != 1 {
		return nil
	}

	rhs, rhsOk := decl.Rhs[0].(*ast.BinaryExpr)
	if !rhsOk {
		return nil
	}

	res, err := cc.isTimeGreaterThanMin(rhs)
	if err == nil {
		if !res {
			return errCalcLessThanMin
		}
	}

	return nil
}

func (cc *connCheck) isValidSelectorExpr(arg *ast.SelectorExpr) error {
	unit, ok := cc.isTimeUnit(arg)
	if !ok {
		return errInvalidParam
	}

	t := calcDuration(unit, 1)

	if t.Seconds() < float64(cc.config.minSeconds) {
		return errCalcLessThanMin
	}

	return nil
}

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
	intVal, hasInt = binaryExprBasicLitToLit(arg)

	// check for CallExpr
	if !hasInt {
		intVal, hasInt = binaryExprCallExprToLit(arg)
	}

	if !hasInt {
		return false, errNoCalc
	}

	t := calcDuration(unit, intVal)

	return t.Seconds() >= float64(cc.config.minSeconds), nil
}

// isValidUnaryExpr checks if the unary expression has a subtraction operator.
// This indicates that the connection should not be closed.
// https://pkg.go.dev/database/sql#DB.SetConnMaxLifetime
func (cc *connCheck) isValidUnaryExpr(arg *ast.UnaryExpr) bool {
	return arg.Op == token.SUB
}

// isValidBasicLit checks if the basic literal is larger than 0 which would
// could indicate that the user forgot to add a time unit as the duration is in
// nano seconds.
func (cc *connCheck) isValidBasicLit(arg *ast.BasicLit) bool {
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

// isValidBinaryExpr checks if the binary expression contains a time unit which
// would indicate that the duration is set correctly.
func (cc *connCheck) isValidBinaryExpr(arg *ast.BinaryExpr) bool {
	hasUnit := false

	if arg.Op == token.MUL {
		_, ok := cc.isTimeUnit(arg.X)
		if ok {
			hasUnit = true
		}

		_, ok = cc.isTimeUnit(arg.Y)
		if ok {
			hasUnit = true
		}
	}

	return hasUnit
}

// isValidCallExpr checks if the call expression is a call to time.Duration
// which could indicate that a time unit is missing.
func (cc *connCheck) isValidCallExpr(arg *ast.CallExpr) error {
	if isCallTimeDuration(arg) {
		return errMissingUnit
	}

	return errPotentialMissingUnit
}

func isCallTimeDuration(arg *ast.CallExpr) bool {
	selector, selectorOk := arg.Fun.(*ast.SelectorExpr)

	if !selectorOk || selector.X == nil {
		return false
	}

	ident, identOk := selector.X.(*ast.Ident)
	if !identOk {
		return false
	}

	return ident.Name == "time" && selector.Sel.Name == "Duration"
}

// isTimeUnit checks if the type is a time unit and is one of the validUnits /
// validTimeUnits.
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

		if ident.Name == "time" && slices.Contains(cc.config.validUnits.slice, selector.Sel.Name) {
			return selector.Sel.Name, true
		}
	default:
		return "", false
	}

	return "", false
}
