package templates

const EndpointsBase = `
package middlewares

import (
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"

	"{{.ImportPath -}} /svc"
)

// WrapEndpoints accepts the service's entire collection of endpoints, so that a
// set of middlewares can be wrapped around every middleware (e.g., access
// logging and instrumentation), and others wrapped selectively around some
// endpoints and not others (e.g., endpoints requiring authenticated access).
// Note that the final middleware wrapped will be the outermost middleware
// (i.e. applied first)
func WrapEndpoints(in svc.Endpoints) svc.Endpoints {

	// Pass in the middlewares you want applied to every endpoint.
	// optionally pass in handlers by name that you want to be excluded
	// e.g.
	// in.WrapAllExcept(authMiddleware, "Status", "Ping")

	// Pass in LabeledMiddlewares you want applied to every endpoint.
	// These middlewares get passed the handlers name as their first argument when applied.
	// This can be used to write generic metric gathering middlewares that can
	// report the handler name for free.
	// in.WrapAllLabeledExcept(errCounter(statsdCounter), "Status", "Ping")

	// How to apply a middleware to a single endpoint.
	// in.ExampleEndpoint = authMiddleware(in.ExampleEndpoint)

	return in
}

// errCounter is a LabeledMiddleware, when applied with WrapAllLabeledExcept
// name will be populated with the handler name, and such this middleware will
// report errors to the metric provider with the handler name. Feel free to
// remove this example middleware
func errorCounter(errCount metrics.Counter) svc.LabeledMiddleware {
	return func(name string, in endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := in(ctx, req)
			if err != nil {
				errCount.With("handler", name).Add(1)
			}
			return resp, err
		}
	}
}
`
