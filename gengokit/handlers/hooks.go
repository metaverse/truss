package handlers

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"

	"github.com/Unity-Technologies/truss/gengokit"
	"github.com/Unity-Technologies/truss/gengokit/handlers/templates"
)

const HookPath = "handlers/hooks.gotemplate"

// NewHook returns a new HookRender
func NewHook(prev io.Reader) (gengokit.Renderable, error) {
	h := new(HookRender)
	if prev != nil {
		h.fset = token.NewFileSet()
		var err error
		h.ast, err = parser.ParseFile(h.fset, "", prev, parser.ParseComments)
		if err != nil {
			return nil, err
		}
	}
	return h, nil
}

type HookRender struct {
	fset *token.FileSet
	ast  *ast.File
}

// Render will return the existing file if it exists, otherwise it will return
// a brand new copy from the template.
func (h *HookRender) Render(path string, _ *gengokit.Data) (io.Reader, error) {
	code := new(bytes.Buffer)
	if h.ast == nil {
		code.WriteString(templates.HookHead)
		for _, hd := range templates.Hooks {
			code.WriteString(hd.Code)
		}
		return code, nil
	}

	// Note which hooks do not need to be added:
	for _, d := range h.ast.Decls {
		switch v := d.(type) {
		case *ast.FuncDecl:
			for _, h := range templates.Hooks {
				if h.Name == v.Name.Name {
					h.Code = ""
					break
				}
			}
		}
	}

	// Place to collect code for any missing hooks:
	extra := new(bytes.Buffer)

	// Add missing imports needed for hooks that will be added:
	included := map[string]bool{}
	for _, i := range h.ast.Imports {
		included[i.Path.Value] = true
	}
	var imp *ast.GenDecl
	for _, hd := range templates.Hooks {
		if "" == hd.Code {
			continue
		}
		// Add source code for this missing hook:
		extra.WriteString(hd.Code)

		for _, i := range hd.Imports {
			i = "\"" + i + "\""
			if included[i] {
				continue
			}
			included[i] = true
			if imp == nil {
				if len(h.ast.Decls) < 1 {
					return nil, fmt.Errorf("No import() statement in %s", path)
				}
				var ok bool
				if imp, ok = h.ast.Decls[0].(*ast.GenDecl); !ok {
					return nil, fmt.Errorf("First import in %s lacks parens", path)
				}
			}
			imp.Specs = append(imp.Specs, &ast.ImportSpec{
				Path: &ast.BasicLit{Kind: token.STRING, Value: i},
			})
		}
	}

	if err := printer.Fprint(code, h.fset, h.ast); nil != err {
		return nil, err
	}
	code.ReadFrom(extra)

	return code, nil
}
