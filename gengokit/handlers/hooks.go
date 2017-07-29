package handlers

import (
	"io"
	"strings"

	"github.com/TuneLab/truss/gengokit"
	"github.com/TuneLab/truss/gengokit/handlers/templates"
	"github.com/TuneLab/truss/kit"
)

const HookPath = "handlers/hooks.go"

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
func (h *HookRender) Render(_ string, data *gengokit.Data) (io.Reader, error) {
	if h.prev == nil {
		return strings.NewReader(templates.Hook[kit.Version]["Hook"]), nil
	} else {
		return h.prev, nil
	}
}
