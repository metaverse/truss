package test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	// 3d Party
	"golang.org/x/net/context"

	pb "github.com/TuneLab/truss/cmd/_integration-tests/transport/transportpermutations-service"
	httpclient "github.com/TuneLab/truss/cmd/_integration-tests/transport/transportpermutations-service/svc/client/http"

	"github.com/TuneLab/truss/cmd/_integration-tests/transport/transportpermutations-service/handlers"
	"github.com/TuneLab/truss/cmd/_integration-tests/transport/transportpermutations-service/middlewares"
	"github.com/TuneLab/truss/cmd/_integration-tests/transport/transportpermutations-service/svc"

	httptransport "github.com/go-kit/kit/transport/http"
)

var assignval interface{}

// See how fast it is to make a full RPC complete with actual (local) HTTP connection
func BenchmarkGetWithQueryClient(b *testing.B) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360

	svchttp, err := httpclient.New(httpAddr, httpclient.CtxValuesToSend("request-url", "transport"))
	if err != nil {
		b.Fatalf("failed to create httpclient: %q", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := svchttp.GetWithQuery(context.Background(), &req)
		if err != nil {
			b.Fatalf("httpclient returned error: %q", err)
		}
		assignval = resp
	}
}

type ServerTestData struct {
	r        *http.Request
	recorder *httptest.ResponseRecorder
}

// Instead of actually sending a request over the network, instead we construct
// the service endpoints within this function and call it's method directly.
func BenchmarkGetWithQueryClient_NoNetwork(b *testing.B) {
	// Create the server for http transport, but without actually having it
	// serve HTTP requests. Instead we're going to pass it a pre-constructed
	// HTTP request directly.
	var service pb.TransportPermutationsServer
	{
		service = handlers.NewService()
		// Wrap Service with middlewares. See middlewares/service.go
		service = middlewares.WrapService(service)
	}
	var getwithqueryEndpoint = svc.MakeGetWithQueryEndpoint(service)
	endpoints := svc.Endpoints{
		GetWithQueryEndpoint: getwithqueryEndpoint,
	}
	ctx := context.WithValue(context.Background(), "request-url", "/getwithquery")
	ctx = context.WithValue(ctx, "transport", "HTTPJSON")
	server := httptransport.NewServer(
		// This is definitely a hack
		ctx,
		endpoints.GetWithQueryEndpoint,
		svc.DecodeHTTPGetWithQueryZeroRequest,
		svc.EncodeHTTPGenericResponse,
	)
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360

	var testData []ServerTestData
	for i := 0; i < b.N; i++ {
		httpreq, _ := http.NewRequest("GET", "http://localhost/getwithquery", strings.NewReader(string("")))
		httpclient.EncodeHTTPGetWithQueryZeroRequest(context.Background(), httpreq, &req)
		resprecorder := httptest.NewRecorder()

		item := ServerTestData{
			r:        httpreq,
			recorder: resprecorder,
		}

		testData = append(testData, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.ServeHTTP(testData[i].recorder, testData[i].r)
	}
}

// Benchmark the speed of encoding an HTTP request
func BenchmarkClientEncoding(b *testing.B) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	for i := 0; i < b.N; i++ {
		httpreq, _ := http.NewRequest("GET", "http://localhost/getwithquery", strings.NewReader(string("")))
		httpclient.EncodeHTTPGetWithQueryZeroRequest(context.Background(), httpreq, &req)
	}
}

// This provides a baseline "minimum speed" benchmark
func BenchmarkAddition(b *testing.B) {
	A := "12"
	B := "45360"

	for i := 0; i < b.N; i++ {
		a, _ := strconv.Atoi(A)
		b, _ := strconv.Atoi(B)

		assignval = a + b
	}
}
