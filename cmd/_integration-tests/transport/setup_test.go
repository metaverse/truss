package test

import (
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	// Go Kit
	"github.com/go-kit/kit/log"

	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transportpermutations-service"
	svc "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transportpermutations-service/generated"
	handler "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transportpermutations-service/handlers/server"
)

func TestMain(m *testing.M) {
	var logger log.Logger
	logger = log.NewNopLogger()

	var service pb.TransportPermutationsServer
	{
		service = handler.NewService()
	}

	// Endpoint domain.
	getWithQueryE := svc.MakeGetWithQueryEndpoint(service)
	getWithRepeatedQueryE := svc.MakeGetWithRepeatedQueryEndpoint(service)
	postWithNestedMessageBodyE := svc.MakePostWithNestedMessageBodyEndpoint(service)
	ctxToCtxE := svc.MakeCtxToCtxEndpoint(service)
	getWithCapsPathE := svc.MakeGetWithCapsPathEndpoint(service)
	errorRPCE := svc.MakeErrorRPCEndpoint(service)

	endpoints := svc.Endpoints{
		GetWithQueryEndpoint:              getWithQueryE,
		GetWithRepeatedQueryEndpoint:      getWithRepeatedQueryE,
		PostWithNestedMessageBodyEndpoint: postWithNestedMessageBodyE,
		CtxToCtxEndpoint:                  ctxToCtxE,
		GetWithCapsPathEndpoint:           getWithCapsPathE,
		ErrorRPCEndpoint:                  errorRPCE,
	}

	ctx := context.Background()

	// http test server
	h := svc.MakeHTTPHandler(ctx, endpoints, logger)
	httpTestServer := httptest.NewServer(h)

	// grpc test server
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	s := grpc.NewServer()
	gs := svc.MakeGRPCServer(ctx, endpoints)
	pb.RegisterTransportPermutationsServer(s, gs)
	go s.Serve(ln)

	httpAddr = httpTestServer.URL
	grpcAddr = ":" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	os.Exit(m.Run())
}
