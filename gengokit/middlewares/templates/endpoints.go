package templates

const EndpointsBase = `
package middlewares

import (
	"github.com/go-kit/kit/endpoint"
)

func InjectEndpointMiddlewares(in endpoint.Endpoint) endpoint.Endpoint {
	return in
}
`
