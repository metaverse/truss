package handlers

import (
	"bytes"
	"io"

	"github.com/Unity-Technologies/truss/gengokit"
	"github.com/Unity-Technologies/truss/gengokit/handlers/templates"
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

// Render will return the existing file if it exists, otherwise it will return
// a brand new copy from the template.
func (h *HookRender) Render(_ string, _ *gengokit.Data) (io.Reader, error) {
	if h.prev != nil {
		return h.prev, nil
	}
	code := new(bytes.Buffer)
	code.WriteString(templates.HookHead)
	for _, hd := range templates.Hooks {
		code.WriteString(hd.Code)
	}

	return code, nil
}
