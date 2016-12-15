package templates

const EndpointsBase = `
package middlewares

import (
	"github.com/go-kit/kit/endpoint"
	svc "{{.ImportPath -}} /generated"
)

// WrapEndpoint will be called individually for all endpoints defined in
// the service. Implement this with the middlewares you want applied to
// every endpoint.
func WrapEndpoint(in endpoint.Endpoint) endpoint.Endpoint {
	return in
}

// WrapEndpoints takes the service's entire collection of endpoints. This
// function can be used to apply middlewares selectively to some endpoints,
// but not others, like protecting some endpoints with authentication.
func WrapEndpoints(in svc.Endpoints) svc.Endpoints {
	return in
}
`
