package templates

const EndpointsBase = `
package middlewares

import (
	svc "{{.ImportPath -}} /generated"
)

// WrapEndpoints accepts the service's entire collection of endpoints, so that a
// set of middlewares can be wrapped around every middleware (e.g., access
// logging and instrumentation), and others wrapped selectively around some
// endpoints and not others (e.g., endpoints requiring authenticated access).
func WrapEndpoints(in svc.Endpoints) svc.Endpoints {

	// Pass in the middlewares you want applied to every endpoint.
	in.WrapAll(/* ...endpoint.Middleware */)

	// How to apply a middleware selectively.
	// in.ExampleEndpoint = authMiddleware(in.ExampleEndpoint)

	return in
}
`
