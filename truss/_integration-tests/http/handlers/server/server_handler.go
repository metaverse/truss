package handler

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	_ "errors"
	_ "time"

	"golang.org/x/net/context"

	_ "github.com/go-kit/kit/log"
	_ "github.com/go-kit/kit/metrics"

	pb "github.com/TuneLab/go-truss/truss/_integration-tests/http/httptest-service"
)

// NewService returns a naïve, stateless implementation of Service.
func NewService() Service {
	return httptestService{}
}

type httptestService struct{}

// GetWithQuery implements Service.
func (s httptestService) GetWithQuery(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}

	return &response, nil
}

// GetWithRepeatedQuery implements Service.
func (s httptestService) GetWithRepeatedQuery(ctx context.Context, in *pb.GetWithRepeatedQueryRequest) (*pb.GetWithRepeatedQueryResponse, error) {
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
func (s httptestService) PostWithNestedMessageBody(ctx context.Context, in *pb.PostWithNestedMessageBodyRequest) (*pb.PostWithNestedMessageBodyResponse, error) {
	response := pb.PostWithNestedMessageBodyResponse{
		V: in.NM.A + in.NM.B,
	}
	return &response, nil
}

// CtxtToCtxtViaHTTPHeader implements Service.
func (s httptestService) CtxToCtxViaHTTPHeader(ctx context.Context, in *pb.HeaderRequest) (*pb.HeaderResponse, error) {
	var resp pb.HeaderResponse
	val := ctx.Value(in.HeaderKey)

	if v, ok := val.(string); ok {
		resp.V = v
	} else if val == nil {
		resp.V = "CONTEXT VALUE FOR KEY IS NILL"
	} else {
		resp.V = "CONTEXT VALUE FOR KEY IS NON STRING"
	}

	return &resp, nil
}

type Service interface {
	GetWithQuery(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error)
	GetWithRepeatedQuery(ctx context.Context, in *pb.GetWithRepeatedQueryRequest) (*pb.GetWithRepeatedQueryResponse, error)
	PostWithNestedMessageBody(ctx context.Context, in *pb.PostWithNestedMessageBodyRequest) (*pb.PostWithNestedMessageBodyResponse, error)
	CtxToCtxViaHTTPHeader(ctx context.Context, in *pb.HeaderRequest) (*pb.HeaderResponse, error)
}
