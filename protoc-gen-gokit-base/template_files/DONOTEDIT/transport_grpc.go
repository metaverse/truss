{{ with $templateExecutor := .}}
{{ with $GeneratedImport := $templateExecutor.GeneratedImport}}
{{ with $strings := $templateExecutor.Strings}}
{{ with $Service := $templateExecutor.Service}}
package addsvc

// This file provides server-side bindings for the gRPC transport.
// It utilizes the transport/grpc.Server.

import (
	stdopentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	// This Service
	"{{$GeneratedImport -}} /pb"
)


// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(ctx context.Context, endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger) pb.AddServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		// {{ call $strings.ToLower $Service.GetName }}
	{{range $i := $Service.Methods}}
		{{call $strings.ToLower $i.GetName}}: grpctransport.NewServer(
			ctx,
			endpoints.{{$i.GetName}}Endpoint,
			DecodeGRPC{{$i.GetName}}Request,
			EncodeGRPC{{$i.GetName}}Response,
			append(options,grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "{{$i.GetName}}", logger)))...,
		),
	{{- end}}
	}
}

type grpcServer struct {
{{range $i := $Service.Methods}}
	{{call $strings.ToLower $i.GetName}}   grpctransport.Handler
{{- end}}
}

// Methods
{{range $i := $Service.Methods}}
func (s *grpcServer) {{$i.GetName}}(ctx context.Context, req *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error) {
	_, rep, err := s.{{call $strings.ToLower $i.GetName}}.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.{{$i.ResponseType.GetName}}), nil
}
{{end}}

// Server Decode
{{range $i := $Service.Methods}}
// DecodeGRPC{{$i.GetName}}Request is a transport/grpc.DecodeRequestFunc that converts a
// gRPC {{call $strings.ToLower $i.GetName}} request to a user-domain {{call $strings.ToLower $i.GetName}} request. Primarily useful in a server.
func DecodeGRPC{{$i.GetName}}Request(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(pb.{{$i.RequestType.GetName}})
	return req, nil
}
{{end}}


// Client Decode
{{range $i := $Service.Methods}}
// DecodeGRPC{{$i.GetName}}Response is a transport/grpc.DecodeResponseFunc that converts a
// gRPC {{call $strings.ToLower $i.GetName}} reply to a user-domain {{call $strings.ToLower $i.GetName}} response. Primarily useful in a client.
func DecodeGRPC{{$i.GetName}}Response(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.{{$i.ResponseType.GetName}})
	return reply, nil
}
{{end}}

// Server Encode
{{range $i := $Service.Methods}}
// EncodeGRPC{{$i.GetName}}Response is a transport/grpc.EncodeResponseFunc that converts a
// user-domain {{call $strings.ToLower $i.GetName}} response to a gRPC {{call $strings.ToLower $i.GetName}} reply. Primarily useful in a server.
func EncodeGRPC{{$i.GetName}}Response(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.{{$i.ResponseType.GetName}})
	return resp, nil
}
{{end}}


// Client Decode
{{range $i := $Service.Methods}}
// EncodeGRPC{{$i.GetName}}Request is a transport/grpc.EncodeRequestFunc that converts a
// user-domain {{call $strings.ToLower $i.GetName}} request to a gRPC {{call $strings.ToLower $i.GetName}} request. Primarily useful in a client.
func EncodeGRPC{{$i.GetName}}Request(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(pb.{{$i.RequestType.GetName}})
	return req, nil
}
{{end}}
{{end}}
{{end}}
{{end}}
{{end}}
