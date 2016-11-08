// package handler parses service handlers and add/removes exported methods to
// compile with the definition service's rpcs
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

	// Will be removed when cliclient is fully generated
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
)

// NewService is an exported func that creates a new service
// it will not be defined in the service definition but is required
const ignoredFunc = "NewService"

// New returns a truss.Renderable capable of updating server or cli-client handlers
// New should be passed the previous version of the server or cli-client handler to parse
func New(svc *svcdef.Service, prev io.Reader) (gengokit.Renderable, error) {
	var h handler
	log.WithField("Service Methods", len(svc.Methods)).Debug("Handler being created")
	mMap := make(map[string]*svcdef.ServiceMethod, len(svc.Methods))
	for _, m := range svc.Methods {
		mMap[m.Name] = m
	}
	h.mMap = mMap
	h.service = svc

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

type handler struct {
	fset    *token.FileSet
	service *svcdef.Service
	mMap    methodMap
	ast     *ast.File
}

type cliHandlerExecutor struct {
	handlerExecutor
	ClientArgs *clientarggen.ClientServiceArgs
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
	if h.ast == nil {
		return h.renderFirst(f, te)
	}

	// Remove exported methods not defined in service definition
	// and remove methods defined in the previous file from methodMap
	log.WithField("Service Methods", len(h.mMap)).Debug("Before prune")
	h.ast.Decls = h.mMap.pruneDecls(h.ast.Decls)
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

	// render the server or client for all methods not already defined
	var newCode io.Reader
	switch f {
	case "NAME-service/handlers/server/server_handler.gotemplate":
		newCode, err = applyTemplate(serverTempl, "ServerTemplate", ex)
	default:
		return nil, errors.Errorf("cannot render unknown file: %q", f)
	}

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
func (m methodMap) pruneDecls(decls []ast.Decl) []ast.Decl {
	var newDecls []ast.Decl
	for _, d := range decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if ok := m.isValidFunc(x); ok == true {
				newDecls = append(newDecls, d)
				delete(m, x.Name.String())
			}
		default:
			newDecls = append(newDecls, d)
		}

	}
	return newDecls
}

// keepCurrentFunc returns true if f is unexported OR if it exists in m.
func (m methodMap) isValidFunc(f *ast.FuncDecl) bool {
	name := f.Name.String()
	log.WithField("Func", name).
		Debug("Examining function")

	if !ast.IsExported(name) {
		log.WithField("Func", name).
			Debug("Unexported function; ignoring")
		return true
	}

	v := m[name]
	if v == nil && name != ignoredFunc {
		log.WithField("Method", name).
			Info("Method does not exist in service definition as an rpc")
		return false
	}

	log.WithField("Func", name).
		Debug("Method already exists in service defintion; ignoring")

	return true
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
