{{ with $templateExecutor := .}}
{{ with $GeneratedImport := $templateExecutor.GeneratedImport}}
{{ with $HandlerImport := $templateExecutor.HandlerImport}}
{{ with $strings := $templateExecutor.Strings}}
{{ with $Service := $templateExecutor.Service}}
// Package grpc provides a gRPC client for the add service.
package grpc

import (
	"time"

	jujuratelimit "github.com/juju/ratelimit"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	// This Service
	handler "{{$HandlerImport -}} /server"
	addsvc "{{$GeneratedImport -}}"
	pb "{{$GeneratedImport -}} /pb"
)

// New returns an AddService backed by a gRPC client connection. It is the
// responsibility of the caller to dial, and later close, the connection.
func New(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) handler.Service {
	// We construct a single ratelimiter middleware, to limit the total outgoing
	// QPS from this client to all methods on the remote instance. We also
	// construct per-endpoint circuitbreaker middlewares to demonstrate how
	// that's done, although they could easily be combined into a single breaker
	// for the entire remote instance, too.

	limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))

{{range $i := $Service.Methods}}
	var {{call $strings.ToLower $i.GetName}}Endpoint endpoint.Endpoint
	{
		{{call $strings.ToLower $i.GetName}}Endpoint = grpctransport.NewClient(
			conn,
			"{{$Service.GetName}}",
			"{{$i.GetName}}",
			addsvc.EncodeGRPC{{$i.GetName}}Request,
			addsvc.DecodeGRPC{{$i.GetName}}Response,
			pb.{{$i.ResponseType.GetName}}{},
			grpctransport.ClientBefore(opentracing.FromGRPCRequest(tracer, "{{$i.GetName}}", logger)),
		).Endpoint()
		{{call $strings.ToLower $i.GetName}}Endpoint = opentracing.TraceClient(tracer, "{{$i.GetName}}")({{call $strings.ToLower $i.GetName}}Endpoint)
		{{call $strings.ToLower $i.GetName}}Endpoint = limiter({{call $strings.ToLower $i.GetName}}Endpoint)
		{{call $strings.ToLower $i.GetName}}Endpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "{{$i.GetName}}",
			Timeout: 30 * time.Second,
		}))({{call $strings.ToLower $i.GetName}}Endpoint)
	}
{{end}}

	return addsvc.Endpoints{
	{{range $i := $Service.Methods}}
		{{$i.GetName}}Endpoint:    {{call $strings.ToLower $i.GetName}}Endpoint,
	{{- end}}
	}
}
{{end}}
{{end}}
{{end}}
{{end}}
{{end}}
