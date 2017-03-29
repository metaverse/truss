package handler

import (
	"golang.org/x/net/context"

	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service"
)

// NewService returns a naïve, stateless implementation of Service.
func NewService() pb.MiddlewaresTestServer {
	return middlewarestestService{}
}

type middlewarestestService struct{}

// AlwaysWrapped implements Service.
func (s middlewarestestService) AlwaysWrapped(ctx context.Context, in *pb.Empty) (*pb.WrapAllExceptTest, error) {
	var resp pb.WrapAllExceptTest

	always := ctx.Value("Always")
	if a, ok := always.(bool); ok {
		resp.Always = a
	}
	notSometimes := ctx.Value("NotSometimes")
	if ns, ok := notSometimes.(bool); ok {
		resp.NotSometimes = ns
	}

	return &resp, nil
}

// SometimesWrapped implements Service.
func (s middlewarestestService) SometimesWrapped(ctx context.Context, in *pb.Empty) (*pb.WrapAllExceptTest, error) {
	var resp pb.WrapAllExceptTest

	always := ctx.Value("Always")
	if a, ok := always.(bool); ok {
		resp.Always = a
	}
	notSometimes := ctx.Value("NotSometimes")
	if ns, ok := notSometimes.(bool); ok {
		resp.NotSometimes = ns
	}

	return &resp, nil
}
