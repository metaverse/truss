package test

import (
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/net/context"

	// Go Kit
	"github.com/go-kit/kit/log"

	svc "github.com/TuneLab/go-truss/truss/_integration-tests/transport/transport-service/generated"
	handler "github.com/TuneLab/go-truss/truss/_integration-tests/transport/transport-service/handlers/server"
)

var httpTestServer *httptest.Server

func TestMain(m *testing.M) {
	var logger log.Logger
	logger = log.NewNopLogger()

	var service handler.Service
	{
		service = handler.NewService()
	}

	// Endpoint domain.
	getWithQueryE := svc.MakeGetWithQueryEndpoint(service)
	getWithRepeatedQueryE := svc.MakeGetWithRepeatedQueryEndpoint(service)
	postWithNestedMessageBodyE := svc.MakePostWithNestedMessageBodyEndpoint(service)
	ctxToCtxE := svc.MakeCtxToCtxEndpoint(service)

	endpoints := svc.Endpoints{
		GetWithQueryEndpoint:              getWithQueryE,
		GetWithRepeatedQueryEndpoint:      getWithRepeatedQueryE,
		PostWithNestedMessageBodyEndpoint: postWithNestedMessageBodyE,
		CtxToCtxEndpoint:                  ctxToCtxE,
	}

	ctx := context.Background()

	h := svc.MakeHTTPHandler(ctx, endpoints, logger)

	httpTestServer = httptest.NewServer(h)

	os.Exit(m.Run())
}
