package conncheck

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	pkgsDefault              = []string{"database/sql", "gorm.io/gorm", "github.com/jmoiron/sqlx"}
	validUnitsDefault        = []string{"Second", "Minute", "Hour"}
	minSecondsDefault uint64 = 60
	printASTDefault          = false

	errMissingUnit          = errors.New("missing a valid time unit")
	errOperator             = errors.New("operator is not -")
	errPotentialMissingUnit = errors.New("potentially missing a time unit")
	errCalcLessThanMin      = errors.New("time is less than minimum required")
	errNoCalc               = errors.New("can't calculate time")
)

type Config struct {
	Pkgs       stringSliceValue
	ValidUnits stringSliceValue
	MinSeconds uint64
	printAST   bool
}

type connCheck struct {
	config *Config
}

// NewConncheck creates a new conncheck.
func NewConncheck(config *Config) *connCheck {
	return &connCheck{
		config: config,
	}
}

// flags returns the command line flags and sets the default configuration
// values.
func (cc *connCheck) flags() flag.FlagSet {
	flags := flag.NewFlagSet("", flag.ExitOnError)

	// If we have a config, we're not running from cli so don't use any flags
	if cc.config != nil {
		return *flags
	}

	cc.config = DefaultConfig()

	flags.Var(&cc.config.Pkgs, "packages", "A comma-separated list of packages to check against")
	flags.Var(&cc.config.ValidUnits, "timeunits", "A comma-separated list of time units to validate against")
	flags.Uint64Var(&cc.config.MinSeconds, "minsec", minSecondsDefault, "The minimum seconds for SetConnMaxLifetime")
	flags.BoolVar(&cc.config.printAST, "printast", printASTDefault, "Print AST")

	return *flags
}

// New creates a new conncheck analyzer for golangci-lint.
func New(config *Config) *analysis.Analyzer {
	cc := NewConncheck(config)

	return &analysis.Analyzer{
		Name:             "conncheck",
		Doc:              "Conncheck checks db.SetConnMaxLifetime is set to a reasonable value",
		Requires:         []*analysis.Analyzer{inspect.Analyzer},
		Run:              cc.run,
		Flags:            cc.flags(),
		RunDespiteErrors: true,
	}
}

// DefaultConfig returns the default config for conncheck.
func DefaultConfig() *Config {
	return &Config{
		Pkgs: stringSliceValue{
			slice: pkgsDefault,
		},
		ValidUnits: stringSliceValue{
			slice: validUnitsDefault,
		},
		MinSeconds: minSecondsDefault,
	}
}

// Run runs the conncheck analyzer.
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

		switch selectorX := selector.X.(type) {
		case *ast.Ident:
			if !isIdentTypeDb(pass.TypesInfo.TypeOf(selectorX).String()) {
				fmt.Printf("no pass typesinfo: %v\n", pass.TypesInfo.TypeOf(selectorX).String())
				return
			}
		case *ast.SelectorExpr:
		default:
			return
		}

		cc.process(pass, call, node)
	})

	return nil, nil
}

func isIdentTypeDb(s string) bool {
	match := false

	supportedIdentTypes := []string{
		"*database/sql.DB",
		"*github.com/jmoiron/sqlx.DB",
		"*github.com/upper/db/v4.Session",
	}

	for _, t := range supportedIdentTypes {
		if strings.Contains(t, s) {
			match = true
		}
	}

	return match
}

// process checks if the CallExpr for SetConnMaxLifetime has a valid time
// duration.
func (cc *connCheck) process(pass *analysis.Pass, call *ast.CallExpr, node ast.Node) {
	if cc.config.printAST {
		printAST(node)
	}

	arg := call.Args[0]

	// fmt.Printf("arg: %+v\n", arg)

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
		err := cc.isValidBinaryExpr(arg)
		if err != nil {
			pass.Report(*newDiagnostic(arg, err.Error()))
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
	case *ast.StarExpr:
		err := cc.isValidStarExpr(arg)
		if err != nil {
			pass.Report(*newDiagnostic(arg, err.Error()))
		}
	default:
	}
}

// hasDbObj checks if the package imports one of the packages listed in
// config.Pkgs, otherwise it skips the file.
func (cc *connCheck) hasDbObj(pass *analysis.Pass) bool {
	var dbObj types.Object

	if pass.Pkg == nil {
		return false
	}

	for _, pkg := range pass.Pkg.Imports() {
		// @TODO: switch to slices.Contains once Golangci-lint support Go 1.21
		if sliceContains(cc.config.Pkgs.slice, pkg.Path()) {
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

	switch decl := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
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
	case *ast.Field:
		sel, selOk := decl.Type.(*ast.SelectorExpr)
		if !selOk {
			return nil
		}

		if isSelectorExprTimeDuration(sel) {
			return errPotentialMissingUnit
		}
	}

	return nil
}

// isValidSelectorExpr checks if the selector is a time unit and is one of the
// valid units. It also checks if the time is greater than the minimum required.
func (cc *connCheck) isValidSelectorExpr(arg *ast.SelectorExpr) error {
	unit, ok := cc.isTimeUnit(arg)
	if !ok {
		return errPotentialMissingUnit
	}

	t := calcDuration(unit, 1)

	if t.Seconds() < float64(cc.config.MinSeconds) {
		return errCalcLessThanMin
	}

	return nil
}

// isValidStarExpr checks pointers to time.Duration which could indicate a
// missing time unit
func (cc *connCheck) isValidStarExpr(arg *ast.StarExpr) error {
	return errPotentialMissingUnit
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
// would indicate that the duration is set correctly. It also checks the time
// value is greater than the minimum required.
func (cc *connCheck) isValidBinaryExpr(arg *ast.BinaryExpr) error {
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

	if !hasUnit {
		return errMissingUnit
	}

	res, err := cc.isTimeGreaterThanMin(arg)
	if err == nil {
		if !res {
			return errCalcLessThanMin
		}
	}

	return nil
}

// isValidCallExpr checks if the call expression is a call to time.Duration
// which indicates that a time unit is missing.
func (cc *connCheck) isValidCallExpr(call *ast.CallExpr) error {
	if isCallExprTimeDuration(call) {
		if len(call.Args) > 0 {
			arg := call.Args[0]

			argLit, ok := arg.(*ast.BasicLit)
			if ok && cc.isValidBasicLit(argLit) {
				return nil
			}
		}

		return errMissingUnit
	}

	return errPotentialMissingUnit
}
