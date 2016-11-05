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
	"github.com/TuneLab/go-truss/svcdef"

	// Will be removed when cliclient is fully generated
	"github.com/TuneLab/go-truss/gengokit/clientarggen"
)

const ignoredFunc = "NewService"

// New returns a truss.Renderable capable of updating server or cli-client handlers
// New should be passed the previous version of the server or cli-client handler to parse
func New(svc *svcdef.Service, prev io.Reader) (gengokit.Renderable, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", prev, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	mMap := make(map[string]*svcdef.ServiceMethod, len(svc.Methods))
	for _, m := range svc.Methods {
		mMap[m.Name] = m
	}

	return &handler{
		fset:      fset,
		prevAst:   f,
		methodMap: mMap,
		service:   svc,
	}, nil
}

type handler struct {
	fset      *token.FileSet
	service   *svcdef.Service
	methodMap map[string]*svcdef.ServiceMethod
	prevAst   *ast.File
}

type cliHandlerExecutor struct {
	handlerExecutor
	ClientArgs *clientarggen.ClientServiceArgs
}

type handlerExecutor struct {
	PackageName string
	Methods     []*svcdef.ServiceMethod
}

func (h *handler) Render(f string, te *gengokit.TemplateExecutor) (io.Reader, error) {
	// Remove exported methods not defined in service definition
	// and remove methods defined in the previous file from methodMap
	h.removeUnknownExportedMethods()

	// create a new executor, and add all methods not defined in the previous file
	ex := handlerExecutor{
		PackageName: te.PackageName,
	}

	for k, v := range h.methodMap {
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
		log.Debug("Generating server handlers....")
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
	err := printer.Fprint(code, h.fset, h.prevAst)

	if err != nil {
		return nil, err
	}

	return code, nil
}

func (h handler) removeUnknownExportedMethods() {
	var newDecls []ast.Decl
	for _, d := range h.prevAst.Decls {
		switch x := d.(type) {
		// If it is a function
		case *ast.FuncDecl:
			name := x.Name.String()

			log.WithField("Func", name).
				Debug("Examining function")

			if !x.Name.IsExported() {
				newDecls = append(newDecls, d)
				log.WithField("Func", name).
					Debug("Unexported function; ignoring")
				continue
			}
			// and it is exported
			m := h.methodMap[name]
			// and it is not defined in the definition then remove it
			if m == nil && name != ignoredFunc {
				log.WithField("Method", name).
					Info("Method does not exist in service definition as an rpc")
				continue
			}
			delete(h.methodMap, name)
			newDecls = append(newDecls, d)
			log.WithField("Func", name).
				Debug("Method already exists in service defintion; ignoring")
		default:
			newDecls = append(newDecls, d)
		}

	}
	h.prevAst.Decls = newDecls
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

const clientTempl = `
{{ with $te := .}}
	{{range $i := $te.Methods}}
		// {{$i.Name}} implements Service.
		func {{$i.Name}}({{with index $te.ClientArgs.MethArgs $i.Name}}{{GoName .FunctionArgs}}{{end}}) (*pb.{{GoName $i.RequestType.Name}}, error){
			{{- with $meth := index $te.ClientArgs.MethArgs $i.Name -}}
				{{- range $param := $meth.Args -}}
					{{- if not $param.IsBaseType -}}
						// Add custom business logic for interpreting {{$param.FlagArg}},
					{{- end -}}
				{{- end -}}
			{{- end -}}
			request := pb.{{GoName $i.RequestType.Name}}{
			{{- with $meth := index $te.ClientArgs.MethArgs $i.Name -}}
				{{range $param := $meth.Args -}}
					{{- if $param.IsBaseType}}
						{{GoName $param.Name}} : {{GoName $param.FlagArg}},
					{{- end -}}
				{{end -}}
			{{- end -}}
			}
			return &request, nil
		}
	{{end}}
{{- end}}
`

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
