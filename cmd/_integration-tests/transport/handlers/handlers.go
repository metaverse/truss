package handlers

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	"context"
	"github.com/pkg/errors"
	"net/http"

	pb "github.com/tuneinc/truss/cmd/_integration-tests/transport/proto"
)

// NewService returns a naïve, stateless implementation of Service.
func NewService() pb.TransportPermutationsServer {
	return transportpermutationsService{}
}

type transportpermutationsService struct{}

// GetWithQuery implements Service.
func (s transportpermutationsService) GetWithQuery(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	if rurl := ctx.Value("request-url").(string); rurl != "/getwithquery" {
		panic("Context Value: request-url, expected '/getwithquery' got " + rurl)
	}
	if t := ctx.Value("transport").(string); t != "HTTPJSON" {
		panic("Context Value: transport, expected 'HTTPJSON' got " + t)
	}

	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}

	return &response, nil
}

// GetWithRepeatedQuery implements Service.
func (s transportpermutationsService) GetWithRepeatedQuery(ctx context.Context, in *pb.GetWithRepeatedQueryRequest) (*pb.GetWithRepeatedQueryResponse, error) {
	var out int64

	for _, v := range in.A {
		out = out + v
	}

	response := pb.GetWithRepeatedQueryResponse{
		V: out,
	}

	return &response, nil
}

// GetWithEnumQuery implements Service.
func (s transportpermutationsService) GetWithEnumQuery(ctx context.Context, in *pb.GetWithEnumQueryRequest) (*pb.GetWithEnumQueryResponse, error) {
	response := pb.GetWithEnumQueryResponse{
		Out: in.In,
	}

	return &response, nil
}

// PostWithNestedMessageBody implements Service.
func (s transportpermutationsService) PostWithNestedMessageBody(ctx context.Context, in *pb.PostWithNestedMessageBodyRequest) (*pb.PostWithNestedMessageBodyResponse, error) {
	response := pb.PostWithNestedMessageBodyResponse{
		V: in.NM.A + in.NM.B,
	}
	return &response, nil
}

// CtxToCtx implements Service.
func (s transportpermutationsService) CtxToCtx(ctx context.Context, in *pb.MetaRequest) (*pb.MetaResponse, error) {
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

// GetWithCapsPath implements Service.
func (s transportpermutationsService) GetWithCapsPath(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}

	return &response, nil
}

// GetWithPathParams implements Service.
func (s transportpermutationsService) GetWithPathParams(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}
	return &response, nil
}

// GetWithEnumQuery implements Service.
func (s transportpermutationsService) GetWithEnumPath(ctx context.Context, in *pb.GetWithEnumQueryRequest) (*pb.GetWithEnumQueryResponse, error) {
	response := pb.GetWithEnumQueryResponse{
		Out: in.In,
	}

	return &response, nil
}

// EchoOddNames implements Service.
func (s transportpermutationsService) EchoOddNames(ctx context.Context, in *pb.OddFieldNames) (*pb.OddFieldNames, error) {
	return in, nil
}

var testError error = errors.New("This error should be json over http transport")

// ErrorRPC implements Service.
func (s transportpermutationsService) ErrorRPC(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	return nil, testError
}

// X2AOddRPCName implements Service.
func (s transportpermutationsService) X2AOddRPCName(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	return in, nil
}

// ErrorRPCNonJSONLong implements Service.
func (s transportpermutationsService) ErrorRPCNonJSONLong(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	var resp pb.Empty
	resp = pb.Empty{}
	return &resp, nil
}

// ErrorRPCNonJSON implements Service.
func (s transportpermutationsService) ErrorRPCNonJSON(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	var resp pb.Empty
	resp = pb.Empty{}
	return &resp, nil
}

// ContentTypeTest implements Service.
func (s transportpermutationsService) ContentTypeTest(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	var resp pb.Empty
	resp = pb.Empty{}
	return &resp, nil
}

// StatusCodeAndNilHeaders implements Service.
func (s transportpermutationsService) StatusCodeAndNilHeaders(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	return nil, httpError{errors.New("test error"), http.StatusTeapot, nil}
}

// StatusCodeAndHeaders implements Service.
func (s transportpermutationsService) StatusCodeAndHeaders(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	return nil, httpError{errors.New("test error"), http.StatusTeapot, map[string][]string{
		"Foo":  []string{"Bar"},
		"Test": []string{"A", "B"},
	}}
}

// CustomVerb implements Service
func (s transportpermutationsService) CustomVerb(ctx context.Context, in *pb.GetWithQueryRequest) (*pb.GetWithQueryResponse, error) {
	response := pb.GetWithQueryResponse{
		V: in.A + in.B,
	}
	return &response, nil
}
