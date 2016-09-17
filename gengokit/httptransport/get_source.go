package httptransport

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
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
		switch decs.(type) {
		case *ast.FuncDecl:
			f := decs.(*ast.FuncDecl)
			if f.Name.String() == funcName {
				fAst = f
			}
		}
	}
	code := bytes.NewBuffer(nil)
	err = printer.Fprint(code, fset, fAst)

	if err != nil {
		return "", fmt.Errorf("couldn't print code for func '%v': %v\n", funcName, err)
	}

	return code.String(), nil
}

// AllFuncSourceCode returns the the source code for all the functions defined
// in the same file as the one provided, including the source of the function
// provided.
func AllFuncSourceCode(val interface{}) (string, error) {

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

	// Search ast for all function declarations
	fncSlc := []*ast.FuncDecl{}
	for _, decs := range fileAst.Decls {
		switch decs.(type) {
		case *ast.FuncDecl:
			f := decs.(*ast.FuncDecl)
			fncSlc = append(fncSlc, f)
		}
	}

	rv := ""
	// Append source of each function to rv
	for _, fnc := range fncSlc {
		code := bytes.NewBuffer(nil)
		err = printer.Fprint(code, fset, fnc)

		if err != nil {
			return "", fmt.Errorf("couldn't print code for func '%v': %v\n", fnc.Name.String(), err)
		}
		rv += code.String() + "\n"
	}

	return rv, nil
}
