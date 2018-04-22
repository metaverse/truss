package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	// 3d Party
	"context"

	pb "github.com/tuneinc/truss/cmd/_integration-tests/transport/proto"
	httpclient "github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/svc/client/http"

	"github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/handlers"
	"github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/svc"

	httptransport "github.com/go-kit/kit/transport/http"
)

var assignval interface{}

// See how fast it is to make a full RPC complete with actual (local) HTTP connection
func BenchmarkGetWithQueryClient(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	svchttp, err := httpclient.New(httpAddr, httpclient.CtxValuesToSend("request-url", "transport"))
	if err != nil {
		b.Fatalf("failed to create httpclient: %q", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := pb.GetWithQueryRequest{
			A: r.Int63(),
			B: r.Int63(),
		}
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
		// Wrap Service with middlewares. See handlers/service_middlewares.go
		service = handlers.WrapService(service)
	}
	var getwithqueryEndpoint = svc.MakeGetWithQueryEndpoint(service)
	endpoints := svc.Endpoints{
		GetWithQueryEndpoint: getwithqueryEndpoint,
	}
	ctx := context.WithValue(context.Background(), "request-url", "/getwithquery")
	ctx = context.WithValue(ctx, "transport", "HTTPJSON")
	server := httptransport.NewServer(
		// This is definitely a hack
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

var benchAddr string

// This provides a baseline "minimum speed" benchmark
func BenchmarkAddition(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := http.Get(benchAddr + fmt.Sprintf("/add?A=%d&B=%d", r.Int63(), r.Int63()))
		if err != nil {
			b.Fatalf("httpclient returned error: %q", err)
		}
		ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		assignval = res.Body
	}
}

type tStruct struct {
	foo string
	bar string
	baz int
}

// Called from TestMain
func setupSimpleServer() *http.ServeMux {
	add := func(w http.ResponseWriter, r *http.Request) {
		vs := r.URL.Query()
		a, _ := strconv.Atoi(vs.Get("a"))
		b, _ := strconv.Atoi(vs.Get("b"))
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.Encode(&pb.GetWithQueryResponse{V: int64(a + b)})
	}

	decode := func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}
		var out tStruct
		json.Unmarshal(body, &out)
	}

	encode := func(w http.ResponseWriter, r *http.Request) {
		vs := r.URL.Query()
		baz, _ := strconv.Atoi(vs.Get("baz"))
		in := tStruct{
			foo: vs.Get("foo"),
			bar: vs.Get("bar"),
			baz: baz,
		}
		_, err := json.Marshal(in)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/add", add)
	mux.HandleFunc("/json/decode", decode)
	mux.HandleFunc("/json/encode", encode)

	return mux
}
