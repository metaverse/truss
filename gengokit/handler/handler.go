// package handler parses service handlers and add/removes exported methods to
// comply with the definition service methods
package handler

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	generatego "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/svcdef"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

// NewService is an exported func that creates a new service
// it will not be defined in the service definition but is required
const ignoredFunc = "NewService"
const serverTemplPath = "NAME-service/handlers/server/server_handler.gotemplate"

// New returns a truss.Renderable capable of updating server handlers.
// New should be passed the previous version of the server handler to parse.
func New(svc *svcdef.Service, prev io.Reader, pkgName string) (gengokit.Renderable, error) {
	var h handler
	log.WithField("Service Methods", len(svc.Methods)).Debug("Handler being created")
	h.mMap = newMethodMap(svc.Methods)
	h.service = svc
	h.pkgName = pkgName

	if prev == nil {
		return &h, nil
	}

	h.fset = token.NewFileSet()
	var err error
	if h.ast, err = parser.ParseFile(h.fset, "", prev, parser.ParseComments); err != nil {
		return nil, err
	}

	return &h, nil
}

// methodMap stores all defined service methods by name and is updated to
// remove service methods already in the handler file.
type methodMap map[string]*svcdef.ServiceMethod

func newMethodMap(meths []*svcdef.ServiceMethod) methodMap {
	mMap := make(methodMap, len(meths))
	for _, m := range meths {
		mMap[m.Name] = m
	}
	return mMap
}

type handler struct {
	fset    *token.FileSet
	service *svcdef.Service
	mMap    methodMap
	ast     *ast.File
	pkgName string
}

type handlerExecutor struct {
	PackageName string
	Methods     []*svcdef.ServiceMethod
}

// Render returns a go code server handler that has functions for all
// ServiceMethods in the service definition.
func (h *handler) Render(f string, te *gengokit.TemplateExecutor) (io.Reader, error) {
	if f != serverTemplPath {
		return nil, errors.Errorf("cannot render unknown file: %q", f)
	}
	if h.ast == nil {
		return applyServerTempl(te)
	}

	// Remove exported methods not defined in service definition
	// and remove methods defined in the previous file from methodMap
	log.WithField("Service Methods", len(h.mMap)).Debug("Before prune")
	h.ast.Decls = h.mMap.pruneDecls(h.ast.Decls, te.PackageName)
	log.WithField("Service Methods", len(h.mMap)).Debug("After prune")

	// create a new executor, and add all methods not defined in the previous file
	ex := handlerExecutor{
		PackageName: te.PackageName,
	}

	// If there are no methods to template then exit early
	if len(h.mMap) == 0 {
		return h.buffer()
	}

	for k, v := range h.mMap {
		log.WithField("Method", k).
			Info("Generating handler from rpc definition")
		ex.Methods = append(ex.Methods, v)
	}

	// get the code out of the ast
	code, err := h.buffer()
	if err != nil {
		return nil, err
	}

	// render the server for all methods not already defined
	newCode, err := applyServerMethsTempl(ex)

	if err != nil {
		return nil, err
	}

	if _, err = code.ReadFrom(newCode); err != nil {
		return nil, err
	}

	return code, nil
}

func (h *handler) buffer() (*bytes.Buffer, error) {
	code := bytes.NewBuffer(nil)
	err := printer.Fprint(code, h.fset, h.ast)

	if err != nil {
		return nil, err
	}

	return code, nil
}

// pruneDecls constructs a new []ast.Decls with the exported funcs in decls
// who's names are not keys in methodMap and/or does not have the function
// receiver pkgName + "Service" ("Handler func")  removed.
//
// When a "Handler func" is not removed from decls that funcs name is also
// deleted from methodMap, resulting in a methodMap only containing keys and
// values for functions defined in the service but not in the handler ast.
//
// In addition pruneDecls will update unremoved "Handler func"s input
// paramaters and output results to by the types described in methodMap's
// serviceMethod for that "Handler func".
func (m methodMap) pruneDecls(decls []ast.Decl, pkgName string) []ast.Decl {
	var newDecls []ast.Decl
	for _, d := range decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			// Special case NewService
			if x.Name.Name == ignoredFunc {
				newDecls = append(newDecls, x)
				continue
			}
			if ok := isValidFunc(x, m, pkgName); ok == true {
				name := x.Name.Name
				updateParams(x, m[name])
				updateResults(x, m[name])
				newDecls = append(newDecls, x)
				delete(m, name)
			}
		default:
			newDecls = append(newDecls, d)
		}

	}
	return newDecls
}

// updateParams updates the second param of f to be `X`.(m.RequestType.Name).
// func ProtoMethod(ctx context.Context, *pb.Old) ...-> func ProtoMethod(ctx context.Context, *pb.(m.RequestType.Name))...
func updateParams(f *ast.FuncDecl, m *svcdef.ServiceMethod) {
	if f.Type.Params.NumFields() != 2 {
		log.WithField("Function", f.Name.Name).
			Warn("Function params signature should be func NAME(ctx context.Context, in *pb.TYPE), cannot fix")
		return
	}
	updatePBFieldType(f.Type.Params.List[1].Type, m.RequestType.Name)
}

// updateResults updates the first result of f to be `X`.(m.ResponseType.Name).
// func ProtoMethod(...) (*pb.Old, error) ->  func ProtoMethod(...) (*pb.(m.ResponseType.Name), error)
func updateResults(f *ast.FuncDecl, m *svcdef.ServiceMethod) {
	if f.Type.Results.NumFields() != 2 {
		log.WithField("Function", f.Name.Name).
			Warn("Function results signature should be (*pb.TYPE, error), cannot fix")
		return
	}
	updatePBFieldType(f.Type.Results.List[0].Type, m.ResponseType.Name)
}

// updatePBFieldType updates t if in the form X.Sel/*X.Sel to X.newType/*X.newType.
func updatePBFieldType(t ast.Expr, newType string) {
	// *pb.TYPE -> pb.TYPE
	if ptr, _ := t.(*ast.StarExpr); ptr != nil {
		t = ptr.X
	}
	// pb.TYPE -> TYPE
	if sel, _ := t.(*ast.SelectorExpr); sel != nil {
		//pb.SOMETYPE -> pb.newType
		sel.Sel.Name = newType
	}
}

// isVaidFunc returns fase if f is exported and does no exist in m with
// reciever pkgName + "Service".
func isValidFunc(f *ast.FuncDecl, m methodMap, pkgName string) bool {
	name := f.Name.String()
	if !ast.IsExported(name) {
		log.WithField("Func", name).
			Debug("Unexported function; ignoring")
		return true
	}

	v := m[name]

	if v == nil {
		log.WithField("Method", name).
			Info("Method does not exist in service definition as an rpc; removing")
		return false
	}

	rName := mRecvTypeString(f.Recv)
	if rName != pkgName+"Service" {
		log.WithField("Func", name).WithField("Receiver", rName).
			Info("Func is exported with improper receiver; removing")
		return false
	}

	log.WithField("Func", name).
		Debug("Method already exists in service definition; ignoring")

	return true
}

// mRecvTypeString returns accepts and *ast.FuncDecl.Recv recv, and returns the
// string of the recv type.
// func (s Foo) Test() {} -> "Foo"
func mRecvTypeString(recv *ast.FieldList) string {
	// func NoRecv {}
	if recv == nil ||
		recv.List[0].Type == nil {
		log.Debug("Function has no reciever")
		return ""
	}

	return exprString(recv.List[0].Type)
}

// exprString returns the string representation of
// ast.Expr for function receivers, parameters, and results.
func exprString(e ast.Expr) string {
	var hasPtr string
	// *Foo -> Foo
	if ptr, _ := e.(*ast.StarExpr); ptr != nil {
		hasPtr = "*"
		e = ptr.X
	}
	// *foo.Foo or foo.Foo
	if sel, _ := e.(*ast.SelectorExpr); sel != nil {
		// *foo.Foo -> foo.Foo
		if ptr, _ := e.(*ast.StarExpr); ptr != nil {
			hasPtr = "*"
			e = ptr.X
		}
		// foo.Foo
		if x, _ := sel.X.(*ast.Ident); x != nil {
			return hasPtr + x.Name + "." + sel.Sel.Name
		}
		return ""
	}

	// Foo
	if base, _ := e.(*ast.Ident); base != nil {
		return hasPtr + base.Name
	}

	return ""
}

func applyServerTempl(exec *gengokit.TemplateExecutor) (io.Reader, error) {
	log.Debug("Rendering handler for the first time")
	return applyTemplate(serverTempl, "ServerTempl", exec)
}

func applyServerMethsTempl(exec handlerExecutor) (io.Reader, error) {
	return applyTemplate(serverMethsTempl, "ServerMethsTempl", exec)
}

func applyTemplate(templ string, templName string, exec interface{}) (io.Reader, error) {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"GoName":  generatego.CamelCase,
	}
	codeTemplate, err := template.New(templName).Funcs(funcMap).Parse(templ)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create template")
	}

	outputBuffer := bytes.NewBuffer(nil)
	err = codeTemplate.Execute(outputBuffer, exec)
	if err != nil {
		return nil, errors.Wrap(err, "template error")
	}

	return outputBuffer, nil
}
