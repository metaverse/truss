{{ with $god := .}}
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
	"{{.AbsoluteRelativeImportPath -}} /pb"
)


// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(ctx context.Context, endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger) pb.AddServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		// {{ call .ToLower .Service.GetName }}
	{{range $i := .Service.Methods}}
		{{call $god.ToLower $i.GetName}}: grpctransport.NewServer(
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
{{range $i := .Service.Methods}}
	{{call $god.ToLower $i.GetName}}   grpctransport.Handler
{{- end}}
}

// Methods
{{range $i := .Service.Methods}}
func (s *grpcServer) {{$i.GetName}}(ctx context.Context, req *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error) {
	_, rep, err := s.{{call $god.ToLower $i.GetName}}.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.{{$i.ResponseType.GetName}}), nil
}
{{end}}

// DecodeGRPCSumRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC sum request to a user-domain sum request. Primarily useful in a server.
{{range $i := .Service.Methods}}
func DecodeGRPC{{$i.GetName}}Request(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(pb.{{$i.GetName}}Request)
//	return req.(pb.{{$i.RequestType.GetName}}), nil
	return req, nil
}
{{end}}

// EncodeGRPCSumResponse is a transport/grpc.EncodeResponseFunc that converts a
// user-domain sum response to a gRPC sum reply. Primarily useful in a server.
{{range $i := .Service.Methods}}
func EncodeGRPC{{$i.GetName}}Response(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.{{$i.ResponseType.GetName}})
	return resp, nil
}
{{end}}

// DecodeGRPCSumResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum reply to a user-domain sum response. Primarily useful in a client.
//func DecodeGRPCSumResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	//reply := grpcReply.(*pb.SumReply)
	//return sumResponse{V: int(reply.V), Err: str2err(reply.Err)}, nil
//}

// DecodeGRPCConcatResponse is a transport/grpc.DecodeResponseFunc that converts
// a gRPC concat reply to a user-domain concat response. Primarily useful in a
// client.
//func DecodeGRPCConcatResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	//reply := grpcReply.(*pb.ConcatReply)
	//return concatResponse{V: reply.V, Err: str2err(reply.Err)}, nil
//}

// EncodeGRPCSumRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
//func EncodeGRPCSumRequest(_ context.Context, request interface{}) (interface{}, error) {
	//req := request.(sumRequest)
	//return &pb.SumRequest{A: int64(req.A), B: int64(req.B)}, nil
//}

// EncodeGRPCConcatRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain concat request to a gRPC concat request. Primarily useful in a
// client.
//func EncodeGRPCConcatRequest(_ context.Context, request interface{}) (interface{}, error) {
	//req := request.(concatRequest)
	//return &pb.ConcatRequest{A: req.A, B: req.B}, nil
//}
{{end}}
