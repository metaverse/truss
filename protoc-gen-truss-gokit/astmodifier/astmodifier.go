package astmodifier

import (
	"bytes"
	"go/ast"
	_ "go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
}

type astModifier struct {
	fset             *token.FileSet
	fileAst          *ast.File
	funcIndexer      *functionIndexer
	funcRemover      *functionRemover
	interfaceRemover *interfaceRemover
}

type functionRemover struct {
	functionsToKeep map[string]bool
}

type functionIndexer struct {
	functionIndex map[string]bool
}

type interfaceRemover struct {
	interfaceToRemove string
}

// New returns a new astModifier from a source file which modifies code intelligently
func New(sourcePath string) *astModifier {
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, sourcePath, nil, 0)
	if err != nil {
		log.WithError(err).Fatal("server/service.go could not be parsed by go/parser into AST")
	}

	return &astModifier{
		fset:    fset,
		fileAst: fileAst,
		funcIndexer: &functionIndexer{
			functionIndex: make(map[string]bool),
		},
		funcRemover: &functionRemover{
			functionsToKeep: make(map[string]bool),
		},
		interfaceRemover: &interfaceRemover{
			interfaceToRemove: "",
		},
	}

}

// String returns the current ast as a string
func (a *astModifier) String() string {
	code := bytes.NewBuffer(nil)
	err := printer.Fprint(code, a.fset, a.fileAst)

	if err != nil {
		log.Fatal("Ast could not output code")
	}

	return code.String()
}

// String returns the current ast as a string
func (a *astModifier) Buffer() *bytes.Buffer {
	code := bytes.NewBuffer(nil)
	err := printer.Fprint(code, a.fset, a.fileAst)

	if err != nil {
		log.Fatal("Ast could not output code")
	}

	return code
}

// RemoveFunctionsExecpt takes a []string of function names to keep
// All other functions are removed from source file
func (a *astModifier) RemoveFunctionsExecpt(functionsToKeep []string) {
	a.funcRemover.functionsToKeep = make(map[string]bool)

	for _, fun := range functionsToKeep {
		a.funcRemover.functionsToKeep[fun] = true
	}
	ast.Walk(a.funcRemover, a.fileAst)
	// Clear the functionToKeep map for next run
}

func (a *astModifier) IndexFunctions() map[string]bool {
	a.funcIndexer.functionIndex = make(map[string]bool)
	ast.Walk(a.funcIndexer, a.fileAst)
	// Clear the functionIndex for next run
	return a.funcIndexer.functionIndex
}

func (a *astModifier) RemoveInterface(interfaceToRemove string) {
	a.interfaceRemover.interfaceToRemove = ""
	a.interfaceRemover.interfaceToRemove = interfaceToRemove
	ast.Walk(a.interfaceRemover, a.fileAst)
}

func (v *interfaceRemover) Visit(node ast.Node) ast.Visitor {
	interfaceFound := false
	var declareIndexToDelete int
	if file, ok := node.(*ast.File); ok {
		log.WithField(
			"File Name", file.Name,
		).Debug("AST File")
		for i, decs := range file.Decls {
			switch specDec := decs.(type) {
			case *ast.GenDecl:
				if !interfaceFound {
					for _, spec := range specDec.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
								log.Debug("InterfaceFound")
								_ = interfaceType
								if typeSpec.Name.String() == v.interfaceToRemove {
									declareIndexToDelete = i
									interfaceFound = true
								}
							}
						}
					}
				}
			}
			if interfaceFound {
				file.Decls = append(file.Decls[:declareIndexToDelete], file.Decls[declareIndexToDelete+1:]...)
			}
		}
	}
	return nil
}

func (v *functionIndexer) Visit(node ast.Node) ast.Visitor {
	var funcsToDelete []int
	_ = funcsToDelete
	if file, ok := node.(*ast.File); ok {
		for i, decs := range file.Decls {
			_ = i
			switch specDec := decs.(type) {
			case *ast.FuncDecl:
				v.functionIndex[specDec.Name.String()] = true
			}
		}
	}
	return nil
}

func (v *functionRemover) Visit(node ast.Node) ast.Visitor {
	var funcsToDelete []int
	if file, ok := node.(*ast.File); ok {
		log.WithField(
			"File Name", file.Name,
		).Debug("AST File")

		for i, decs := range file.Decls {
			switch specDec := decs.(type) {
			case *ast.FuncDecl:
				funcName := specDec.Name.String()
				if v.functionsToKeep[funcName] == false {
					log.WithField("Method", funcName).Info("Handler does not exist in proto service description, deleting...")
					funcsToDelete = append(funcsToDelete, i)
				}
			}
		}
		for _, index := range funcsToDelete {
			file.Decls = append(file.Decls[:index], file.Decls[index+1:]...)
		}
	}
	return nil
}
