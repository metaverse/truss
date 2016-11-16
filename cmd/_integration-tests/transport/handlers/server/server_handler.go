package handler

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	"errors"
	"golang.org/x/net/context"

	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transport-service"
)

// NewService returns a naïve, stateless implementation of Service.
func NewService() pb.TransportPermutationsServer {
	return transportService{}
}

type transportService struct{}

// GetWithQuery implements Service.
func (s transportService) GetWithQuery(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}

	return &response, nil
}

// GetWithRepeatedQuery implements Service.
func (s transportService) GetWithRepeatedQuery(ctx context.Context, in *pb.GetWithRepeatedQueryRequest) (*pb.GetWithRepeatedQueryResponse, error) {
	var out int64

	for _, v := range in.A {
		out = out + v
	}

	response := pb.GetWithRepeatedQueryResponse{
		V: out,
	}

	return &response, nil
}

// PostWithNestedMessageBody implements Service.
func (s transportService) PostWithNestedMessageBody(ctx context.Context, in *pb.PostWithNestedMessageBodyRequest) (*pb.PostWithNestedMessageBodyResponse, error) {
	response := pb.PostWithNestedMessageBodyResponse{
		V: in.NM.A + in.NM.B,
	}
	return &response, nil
}

// CtxToCtx implements Service.
func (s transportService) CtxToCtx(ctx context.Context, in *pb.MetaRequest) (*pb.MetaResponse, error) {
	var resp pb.MetaResponse
	val := ctx.Value(in.Key)

	if v, ok := val.(string); ok {
		resp.V = v
	} else if val == nil {
		resp.V = "CONTEXT VALUE FOR KEY IS NILL"
	} else {
		resp.V = "CONTEXT VALUE FOR KEY IS NON STRING"
	}

	return &resp, nil
}

var testError error = errors.New("This error should be json over http transport")

// ErrorRPC implements Service.
func (s transportService) ErrorRPC(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	return nil, testError
}
