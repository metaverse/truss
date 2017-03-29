package test

import (
	"os"
	"testing"

	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service"
	handler "github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service/handlers"
	"github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service/middlewares"
	svc "github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service/svc"
)

var middlewareEndpoints svc.Endpoints

func TestMain(m *testing.M) {

	var service pb.MiddlewaresTestServer
	{
		service = handler.NewService()
	}

	// Endpoint domain.
	alwaysWrapped := svc.MakeAlwaysWrappedEndpoint(service)
	sometimesWrapped := svc.MakeSometimesWrappedEndpoint(service)

	middlewareEndpoints = svc.Endpoints{
		AlwaysWrappedEndpoint:    alwaysWrapped,
		SometimesWrappedEndpoint: sometimesWrapped,
	}

	middlewareEndpoints = middlewares.WrapEndpoints(middlewareEndpoints)

	os.Exit(m.Run())
}
