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
	"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/pb"
)

// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(ctx context.Context, endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger) pb.AddServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		// add

		sum: grpctransport.NewServer(
			ctx,
			endpoints.SumEndpoint,
			DecodeGRPCSumRequest,
			EncodeGRPCSumResponse,
			append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
		concat: grpctransport.NewServer(
			ctx,
			endpoints.ConcatEndpoint,
			DecodeGRPCConcatRequest,
			EncodeGRPCConcatResponse,
			append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Concat", logger)))...,
		),
	}
}

type grpcServer struct {
	sum    grpctransport.Handler
	concat grpctransport.Handler
}

// Methods

func (s *grpcServer) Sum(ctx context.Context, req *pb.SumRequest) (*pb.SumReply, error) {
	_, rep, err := s.sum.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.SumReply), nil
}

func (s *grpcServer) Concat(ctx context.Context, req *pb.ConcatRequest) (*pb.ConcatReply, error) {
	_, rep, err := s.concat.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.ConcatReply), nil
}

// Server Decode

// DecodeGRPCSumRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC sum request to a user-domain sum request. Primarily useful in a server.
func DecodeGRPCSumRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(pb.SumRequest)
	return req, nil
}

// DecodeGRPCConcatRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC concat request to a user-domain concat request. Primarily useful in a server.
func DecodeGRPCConcatRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(pb.ConcatRequest)
	return req, nil
}

// Client Decode

// DecodeGRPCSumResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum reply to a user-domain sum response. Primarily useful in a client.
func DecodeGRPCSumResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.SumReply)
	return reply, nil
}

// DecodeGRPCConcatResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC concat reply to a user-domain concat response. Primarily useful in a client.
func DecodeGRPCConcatResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.ConcatReply)
	return reply, nil
}

// Server Encode

// EncodeGRPCSumResponse is a transport/grpc.EncodeResponseFunc that converts a
// user-domain sum response to a gRPC sum reply. Primarily useful in a server.
func EncodeGRPCSumResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.SumReply)
	return resp, nil
}

// EncodeGRPCConcatResponse is a transport/grpc.EncodeResponseFunc that converts a
// user-domain concat response to a gRPC concat reply. Primarily useful in a server.
func EncodeGRPCConcatResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.ConcatReply)
	return resp, nil
}

// Client Decode

// EncodeGRPCSumRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
func EncodeGRPCSumRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(pb.SumRequest)
	return req, nil
}

// EncodeGRPCConcatRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain concat request to a gRPC concat request. Primarily useful in a client.
func EncodeGRPCConcatRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(pb.ConcatRequest)
	return req, nil
}
