package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type stringSliceValue struct {
	slice []string
}

// Implement the Set method of the flag.Value interface
func (ss *stringSliceValue) Set(s string) error {
	ss.slice = strings.Split(s, ",")
	return nil
}

// Implement the String method of the flag.Value interface
func (ss *stringSliceValue) String() string {
	return strings.Join(ss.slice, ",")
}

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

	_ = ast.Fprint(os.Stdout, nil, node, nil) //nolint:errcheck // ignore

	fmt.Println("--------------")
}

func basicLitValue(arg *ast.BasicLit) (int64, bool) {
	intVal, err := strconv.ParseInt(arg.Value, 10, 64)
	if err != nil {
		return 0, false
	}

	return intVal, true
}
