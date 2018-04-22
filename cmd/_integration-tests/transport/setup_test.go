package test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"google.golang.org/grpc"

	pb "github.com/tuneinc/truss/cmd/_integration-tests/transport/proto"
	handler "github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/handlers"
	svc "github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/svc"
)

func TestMain(m *testing.M) {
	var service pb.TransportPermutationsServer
	{
		service = handler.NewService()
	}

	// Endpoint domain.
	getWithQueryE := svc.MakeGetWithQueryEndpoint(service)
	getWithRepeatedQueryE := svc.MakeGetWithRepeatedQueryEndpoint(service)
	getWithEnumQueryE := svc.MakeGetWithEnumQueryEndpoint(service)
	postWithNestedMessageBodyE := svc.MakePostWithNestedMessageBodyEndpoint(service)
	ctxToCtxE := svc.MakeCtxToCtxEndpoint(service)
	getWithCapsPathE := svc.MakeGetWithCapsPathEndpoint(service)
	getWithPathParamsE := svc.MakeGetWithPathParamsEndpoint(service)
	getWithEnumPathE := svc.MakeGetWithEnumPathEndpoint(service)
	echoOddNamesE := svc.MakeEchoOddNamesEndpoint(service)
	errorRPCE := svc.MakeErrorRPCEndpoint(service)
	errorRPCNonJSONE := svc.MakeErrorRPCNonJSONEndpoint(service)
	errorRPCNonJSONLongE := svc.MakeErrorRPCNonJSONLongEndpoint(service)
	X2AOddRPCNameE := svc.MakeX2AOddRPCNameEndpoint(service)
	contentTypeTestE := svc.MakeContentTypeTestEndpoint(service)
	StatusCodeAndNilHeadersE := svc.MakeStatusCodeAndNilHeadersEndpoint(service)
	StatusCodeAndHeadersE := svc.MakeStatusCodeAndHeadersEndpoint(service)
	CustomVerbE := svc.MakeCustomVerbEndpoint(service)

	endpoints := svc.Endpoints{
		GetWithQueryEndpoint:              getWithQueryE,
		GetWithRepeatedQueryEndpoint:      getWithRepeatedQueryE,
		GetWithEnumQueryEndpoint:          getWithEnumQueryE,
		PostWithNestedMessageBodyEndpoint: postWithNestedMessageBodyE,
		CtxToCtxEndpoint:                  ctxToCtxE,
		GetWithCapsPathEndpoint:           getWithCapsPathE,
		GetWithPathParamsEndpoint:         getWithPathParamsE,
		GetWithEnumPathEndpoint:           getWithEnumPathE,
		EchoOddNamesEndpoint:              echoOddNamesE,
		ErrorRPCEndpoint:                  errorRPCE,
		ErrorRPCNonJSONEndpoint:           errorRPCNonJSONE,
		ErrorRPCNonJSONLongEndpoint:       errorRPCNonJSONLongE,
		X2AOddRPCNameEndpoint:             X2AOddRPCNameE,
		ContentTypeTestEndpoint:           contentTypeTestE,
		StatusCodeAndNilHeadersEndpoint:   StatusCodeAndNilHeadersE,
		StatusCodeAndHeadersEndpoint:      StatusCodeAndHeadersE,
		CustomVerbEndpoint:                CustomVerbE,
	}

	// http test server
	h := svc.MakeHTTPHandler(endpoints)
	httpTestServer := httptest.NewServer(h)

	// grpc test server
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	s := grpc.NewServer()
	gs := svc.MakeGRPCServer(endpoints)
	pb.RegisterTransportPermutationsServer(s, gs)
	go s.Serve(ln)

	httpAddr = httpTestServer.URL
	grpcAddr = ":" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	// Set up a http server that returns non JSON responses
	bmux := http.NewServeMux()
	// Simple non-json response
	bmux.HandleFunc("/error/non/json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(brokenHTTPResponse))
	})
	// Put 16KB of non-json data into the response body
	bmux.HandleFunc("/error/non/json/long", func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 8196*2; i++ {
			w.Write([]byte{byte(i % 256)})
		}
	})
	nonJSONHTTPServer := httptest.NewServer(bmux)
	nonJSONHTTPAddr = nonJSONHTTPServer.URL

	mux := setupSimpleServer()
	benchServer := httptest.NewServer(mux)
	benchAddr = benchServer.URL

	os.Exit(m.Run())
}
