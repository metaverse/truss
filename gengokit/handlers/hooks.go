package handlers

import (
	"bytes"
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
func (h *HookRender) Render(_ string, _ *gengokit.Data) (io.Reader, error) {
	code := new(bytes.Buffer)
	if h.ast == nil {
		code.WriteString(templates.HookHead)
		for _, hd := range templates.Hooks {
			code.WriteString(hd.Code)
		}
		return code, nil
	}

	if err := printer.Fprint(code, h.fset, h.ast); nil != err {
		return nil, err
	}

	return code, nil
}
