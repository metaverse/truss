// Package middlewares renders the service and endpointe middleware files in middlewares/.
package middlewares

import (
	"io"

	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/middlewares/templates"
)

// EndpointsPath is the path to the Endpoints middleware file that package middlewares renders
const EndpointsPath = "middlewares/endpoints.gotemplate"

// ServicePath is the path to the Service middleware file that package middlewares renders
const ServicePath = "middlewares/service.gotemplate"

// New returns a Middleware which can render EndpointsFile and ServiceFile as
// well as read in previous versions of each respective file
func New() *Middlewares {
	var m Middlewares

	return &m
}

// Middlewares satisfies the gengokit.Renderable interface to render
// middlewares, it has methods to load previous versions of the middlewares in
// to update them.
type Middlewares struct {
	// contains unexported fields
	prevEndpoints io.Reader
	prevService   io.Reader
}

// LoadEndpoints loads a previous version of EndpointsFile
func (m *Middlewares) LoadEndpoints(prev io.Reader) {
	m.prevEndpoints = prev
}

// LoadService loads a previous version of ServiceFile
func (m *Middlewares) LoadService(prev io.Reader) {
	m.prevService = prev
}

// Render can render either EndpointsPath or ServicePath. With no previous
// version it renders the templates, if there was a previous version loaded in,
// it passes that through
func (m *Middlewares) Render(path string, data *gengokit.Data) (io.Reader, error) {
	switch path {
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
	return nil, errors.Errorf("cannot render unknown file: %q", path)
}
