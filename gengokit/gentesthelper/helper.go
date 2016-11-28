package gentesthelper

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"reflect"
	"runtime"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// FuncSourceCode returns a string representing the source code of the function
// provided to it.
func FuncSourceCode(val interface{}) (string, error) {
	ptr := reflect.ValueOf(val).Pointer()
	fpath, _ := runtime.FuncForPC(ptr).FileLine(ptr)

	funcName := runtime.FuncForPC(ptr).Name()
	parts := strings.Split(funcName, ".")
	funcName = parts[len(parts)-1]

	// Parse the go file into the ast
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("ERROR: go parser couldn't parse file '%v'\n", fpath)
	}

	// Search ast for function declaration with name of function passed
	var fAst *ast.FuncDecl
	for _, decs := range fileAst.Decls {
		if f, ok := decs.(*ast.FuncDecl); ok && f.Name.String() == funcName {
			fAst = f
			break
		}
	}
	code := bytes.NewBuffer(nil)
	err = printer.Fprint(code, fset, fAst)

	if err != nil {
		return "", fmt.Errorf("couldn't print code for func %q: %v\n", funcName, err)
	}

	return code.String(), nil
}

// DiffStrings returns the line differences of two strings. Useful for
// examining how generated code differs from expected code.
func DiffStrings(a, b string) string {
	t := difflib.UnifiedDiff{
		A:        difflib.SplitLines(a),
		B:        difflib.SplitLines(b),
		FromFile: "A",
		ToFile:   "B",
		Context:  5,
	}
	text, _ := difflib.GetUnifiedDiffString(t)
	return text
}

// DiffGoCode returns normalized versions of inA and inB using the go formatter
// so that differences in indentation or trailing spaces are ignored. A diff of
// inA and inB is also returned.
func DiffGoCode(inA, inB string) (outA, outB, diff string) {
	codeFormat := func(in string) string {
		// Trim starting and ending space so format starts indenting at 0 for
		// both strings
		out := strings.TrimSpace(in)

		// Format code, if we get an error we keep out the same,
		// otherwise we use the formmated version
		outBytes, err := format.Source([]byte(out))
		if err != nil {
			return "FAILED TO FORMAT\n" + out
		} else {
			return string(outBytes)
		}

		return out
	}
	outA = codeFormat(inA)
	outB = codeFormat(inB)
	diff = DiffStrings(
		outA,
		outB,
	)
	return
}

// testFormat takes a string representing golang code and attempts to return a
// formated copy of that code.
func TestFormat(code string) (string, error) {
	formatted, err := format.Source([]byte(code))

	if err != nil {
		return code, err
	}

	return string(formatted), nil
}
