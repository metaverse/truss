package middlewares

import (
	"io"

	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/middlewares/templates"
)

const EndpointsPath = "NAME-service/middlewares/endpoints.gotemplate"
const ServicePath = "NAME-service/middlewares/service.gotemplate"

func New() *Middlewares {
	var m Middlewares

	return &m
}

type Middlewares struct {
	prevEndpoints io.Reader
	prevService   io.Reader
}

func (m *Middlewares) LoadEndpoints(prev io.Reader) {
	m.prevEndpoints = prev
}

func (m *Middlewares) LoadService(prev io.Reader) {
	m.prevService = prev
}

func (m *Middlewares) Render(alias string, data *gengokit.Data) (io.Reader, error) {
	switch alias {
	case EndpointsPath:
		if m.prevEndpoints != nil {
			return m.prevEndpoints, nil
		}
		return data.ApplyTemplate(templates.EndpointsBase, "Endpoint")
	case ServicePath:
		if m.prevService != nil {
			return m.prevService, nil
		}
		return data.ApplyTemplate(templates.ServiceBase, "Service")
	}
	return nil, errors.Errorf("cannot render unknown file: %q", alias)
}
