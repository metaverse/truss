package handler

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	_ "errors"
	_ "time"

	"golang.org/x/net/context"

	_ "github.com/go-kit/kit/log"
	_ "github.com/go-kit/kit/metrics"

	pb "github.com/TuneLab/gob/demo/add/service/DONOTEDIT/pb"
)

// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

// Sum implements Service.
func (s basicService) Sum(ctx context.Context, in pb.SumRequest) (pb.SumReply, error) {
	_ = ctx
	_ = in
	response := pb.SumReply{
		Result: in.A + in.B,
	}
	return response, nil
}

// Concat implements Service.
func (s basicService) Concat(ctx context.Context, in pb.ConcatRequest) (pb.ConcatRequest, error) {
	_ = ctx
	_ = in
	response := pb.ConcatRequest{
	//
	}
	return response, nil
}

type Service interface {
	Sum(ctx context.Context, in pb.SumRequest) (pb.SumReply, error)
	Concat(ctx context.Context, in pb.ConcatRequest) (pb.ConcatRequest, error)
}
