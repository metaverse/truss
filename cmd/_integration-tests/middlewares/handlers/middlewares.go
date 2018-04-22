package handlers

import (
	"context"
	"github.com/go-kit/kit/endpoint"

	svc "github.com/tuneinc/truss/cmd/_integration-tests/middlewares/middlewarestest-service/svc"
	pb "github.com/tuneinc/truss/cmd/_integration-tests/middlewares/proto"
)

// WrapEndpoints accepts the service's entire collection of endpoints, so that a
// set of middlewares can be wrapped around every middleware (e.g., access
// logging and instrumentation), and others wrapped selectively around some
// endpoints and not others (e.g., endpoints requiring authenticated access).
// Note that the final middleware applied will be the outermost middleware
// (i.e. applied first)
func WrapEndpoints(in svc.Endpoints) svc.Endpoints {

	// Pass in the middlewares you want applied to every endpoint.
	// optionally pass in endpoints by name that you want to be excluded
	// e.g.
	// in.WrapAll(authMiddleware, "Status", "Ping")
	in.WrapAllExcept(addBoolToContext("NotSometimes"), "SometimesWrapped")
	in.WrapAllExcept(addBoolToContext("Always"))

	in.WrapAllLabeledExcept(addNameToContext())

	return in
}

func WrapService(in pb.MiddlewaresTestServer) pb.MiddlewaresTestServer {
	return in
}

func addBoolToContext(key string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			ctx = context.WithValue(ctx, key, true)
			return next(ctx, request)
		}
	}
}

func addNameToContext() svc.LabeledMiddleware {
	return func(name string, next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			ctx = context.WithValue(ctx, "handlerName", name)
			return next(ctx, req)
		}
	}
}
