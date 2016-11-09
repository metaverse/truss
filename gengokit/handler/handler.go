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
	templFiles "github.com/TuneLab/go-truss/gengokit/template"
	"github.com/TuneLab/go-truss/svcdef"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

// NewService is an exported func that creates a new service
// it will not be defined in the service definition but is required
const ignoredFunc = "NewService"
const serverTemplPath = "NAME-service/handlers/server/server_handler.gotemplate"

// New returns a truss.Renderable capable of updating server or cli-client handlers
// New should be passed the previous version of the server or cli-client handler to parse
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

// methodMap stores all defined service methods by name
// and is updated to remove service methods already in the handler file
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

func (h *handler) renderFirst(f string, te *gengokit.TemplateExecutor) (io.Reader, error) {
	log.WithField("Template", f).
		Debug("Rendering for the first time from assets")
	t, err := templFiles.Asset(f)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot access template: %q", f)
	}
	return applyTemplate(string(t), f, te)
}

// Render returns
func (h *handler) Render(f string, te *gengokit.TemplateExecutor) (io.Reader, error) {
	if f != serverTemplPath {
		return nil, errors.Errorf("cannot render unknown file: %q", f)
	}
	if h.ast == nil {
		return h.renderFirst(f, te)
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
	newCode, err := applyServerTempl(ex)

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

// pruneDecls constructs a new []ast.Decls with the functions in decls
// who's names are keys in methodMap removed. When a function is removed
// from decls that key is also deleted from methodMap, resulting in a
// methodMap only containing keys and values for functions defined in the
// service but not the handler ast.
func (m methodMap) pruneDecls(decls []ast.Decl, pkgName string) []ast.Decl {
	var newDecls []ast.Decl
	for _, d := range decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if ok := isValidFunc(x, m, pkgName); ok == true {
				newDecls = append(newDecls, d)
				delete(m, x.Name.String())
			}
		default:
			newDecls = append(newDecls, d)
		}

	}
	return newDecls
}

// isVaidFunc returns fase if f is exported and does no exist in m with
// reciever pkgName + "Service"
func isValidFunc(f *ast.FuncDecl, m methodMap, pkgName string) bool {
	name := f.Name.String()
	if !ast.IsExported(name) || name == ignoredFunc {
		log.WithField("Func", name).
			Debug("Unexported or ignored function; ignoring")
		return true
	}

	v := m[name]

	if v == nil {
		log.WithField("Method", name).
			Info("Method does not exist in service definition as an rpc; removing")
		return false
	}

	rName := mRecvName(f.Recv)
	if rName != pkgName+"Service" {
		log.WithField("Func", name).WithField("Receiver", rName).
			Info("Func is exported with improper receiver; removing")
		return false
	}

	log.WithField("Func", name).
		Debug("Method already exists in service definition; ignoring")

	return true
}

func mRecvName(recv *ast.FieldList) string {
	if recv == nil ||
		recv.List[0].Type == nil {
		log.Debug("Function has no reciever")
		return ""
	}

	typ := recv.List[0].Type
	if ptr, _ := typ.(*ast.StarExpr); ptr != nil {
		typ = ptr.X
	}
	if base, _ := typ.(*ast.Ident); base != nil {
		return base.Name
	}

	return ""
}

const serverTempl = `
{{ with $te := .}}
		{{range $i := .Methods}}
		// {{.Name}} implements Service.
		func (s {{$te.PackageName}}Service) {{.Name}}(ctx context.Context, in *pb.{{GoName .RequestType.Name}}) (*pb.{{GoName .ResponseType.Name}}, error){
			var resp pb.{{GoName .ResponseType.Name}}
			resp = pb.{{GoName .ResponseType.Name}}{
				{{range $j := $i.ResponseType.Message.Fields -}}
					// {{GoName $j.Name}}:
				{{end -}}
			}
			return &resp, nil
		} 
		{{end}}
{{- end}}
`

func applyServerTempl(exec handlerExecutor) (io.Reader, error) {
	return applyTemplate(serverTempl, "ServerTempl", exec)
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
