package templates

const EndpointsBase = `
package middlewares

import (
	svc "{{.ImportPath -}} /generated"
)

// WrapEndpoints takes the service's entire collection of endpoints. This
// function can be used to apply middlewares selectively to some endpoints,
// but not others, like protecting some endpoints with authentication.
func WrapEndpoints(in svc.Endpoints) svc.Endpoints {

	// Apply a middleware selectively
	// in.Echo = endpoint.Middleware(in.Echo)


	// Pass in the middlewares you want applied to every endpoint.
	in.WrapAll(/* ...endpoint.Middleware */)

	return in
}
`
