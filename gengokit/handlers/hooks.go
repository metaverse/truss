package handlers

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gochipon/truss/gengokit"
	"github.com/gochipon/truss/gengokit/handlers/templates"
)

const HookPath = "handlers/hooks.gotemplate"

// NewHook returns a new HookRender
func NewHook(prev io.Reader) gengokit.Renderable {
	return &HookRender{
		prev: prev,
	}
}

type HookRender struct {
	prev io.Reader
}

// Render returns an io.Reader with the contents of
// <svcname>/handlers/hooks.go. If hooks.go does not already exist, then it's
// rendered anew from the templates defined in
// 'gengokit/handlers/templates/hook.go'. If hooks.go does exist already, then:
//
//     1. Modify the new code so that it will import
//        "{{.ImportPath}}/svc/server" if it doesn't already.
//     2. Add the InterruptHandler if it doesn't exist already
//     3. Add the SetConfig function if it doesn't exist already
func (h *HookRender) Render(_ string, data *gengokit.Data) (io.Reader, error) {
	if h.prev == nil {
		return data.ApplyTemplate(templates.Hook+templates.HookInterruptHandler+templates.HookSetConfig, "HooksFullTemplate")
	}
	rawprev, err := ioutil.ReadAll(h.prev)
	if err != nil {
		return nil, err
	}
	code := bytes.NewBuffer(rawprev)

	fset := token.NewFileSet()
	past, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	err = addServerImportIfNotPresent(past, data)
	if err != nil {
		return nil, err
	}

	var existingFuncs map[string]bool = map[string]bool{}
	for _, d := range past.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			name := x.Name.Name
			existingFuncs[name] = true
		}
	}
	code = bytes.NewBuffer(nil)
	err = printer.Fprint(code, fset, past)
	if err != nil {
		return nil, err
	}

	// Both of these functions need to be in hooks.go in order for the service to start.
	hookFuncs := map[string]string{
		"InterruptHandler": templates.HookInterruptHandler,
		"SetConfig":        templates.HookSetConfig,
	}

	for name, f := range hookFuncs {
		if _, ok := existingFuncs[name]; !ok {
			code.ReadFrom(strings.NewReader(f))
		}
	}
	return code, nil
}

// addServerImportIfNotPresent ensures that the hooks.go file imports the
// "{{.ImportPath -}} /svc/server" file since the SetConfig function requires
// that import in order to compile. It does this by mutating the handlerfile
// provided as parameter hf in place.
func addServerImportIfNotPresent(hf *ast.File, exec *gengokit.Data) error {
	var imports *ast.GenDecl
	for _, decl := range hf.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			imports = decl.(*ast.GenDecl)
			break
		}
	}

	targetPathTmpl := `"{{.ImportPath -}} /svc"`
	r, err := exec.ApplyTemplate(targetPathTmpl, "ServerPathTempl")
	if err != nil {
		return err
	}
	tmp, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	targetpath := string(tmp)

	for _, spec := range imports.Specs {
		switch spec.(type) {
		case *ast.ImportSpec:
			imp := spec.(*ast.ImportSpec)
			if imp.Path.Value == targetpath {
				return nil
			}
		}
	}

	nimp := ast.ImportSpec{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				&ast.Comment{
					Text: "// This Service",
				},
			},
		},
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: targetpath,
		},
	}
	imports.Specs = append(imports.Specs, &nimp)
	return nil
}
