package handlers

import (
	"github.com/metaverse/truss/gengokit"
	"github.com/metaverse/truss/gengokit/handlers/templates"
	"github.com/pkg/errors"
	"io"
)

const RegisterPath = "handlers/register.gotemplate"

func NewRegister() *Register {
	var r Register
	return &r
}

type Register struct {
	prev io.Reader
}

func (r *Register) Load(prev io.Reader) {
	r.prev = prev
}

func (r *Register) Render(path string, data *gengokit.Data) (io.Reader, error) {
	if path != RegisterPath {
		return nil, errors.Errorf("cannot render unknown file: %q", path)
	}
	if r.prev != nil {
		return r.prev, nil
	}
	return data.ApplyTemplate(templates.Register, "Register")
}
