package test

import (
	"testing"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	// 3d Party
	"golang.org/x/net/context"

	// Go Kit
	"github.com/go-kit/kit/log"

	// This Service
	pb "github.com/TuneLab/go-truss/truss/_integration-tests/http/httptest-service"
	svc "github.com/TuneLab/go-truss/truss/_integration-tests/http/httptest-service/generated"
	httpclient "github.com/TuneLab/go-truss/truss/_integration-tests/http/httptest-service/generated/client/http"
	handler "github.com/TuneLab/go-truss/truss/_integration-tests/http/httptest-service/handlers/server"

	"github.com/pkg/errors"
)

var httpserver *httptest.Server

var _ = io.Copy

func init() {
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

	endpoints := svc.Endpoints{
		GetWithQueryEndpoint:              getWithQueryE,
		GetWithRepeatedQueryEndpoint:      getWithRepeatedQueryE,
		PostWithNestedMessageBodyEndpoint: postWithNestedMessageBodyE,
	}

	ctx := context.Background()

	h := svc.MakeHTTPHandler(ctx, endpoints, logger)
	httpserver = httptest.NewServer(h)
}

func TestGetWithQueryClient(t *testing.T) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	want := req.A + req.B

	svchttp, err := httpclient.New(httpserver.URL)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.GetWithQuery(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != want {
		t.Fatalf("Expect: %d, got %d", want, resp.V)
	}
}

func TestGetWithQueryRequest(t *testing.T) {
	var resp pb.GetWithQueryResponse

	var A, B int64
	A = 12
	B = 45360
	want := A + B

	testHTTP := func(bodyBytes []byte, method, routeFormat string, routeFields ...interface{}) {
		respBytes, err := httpRequestBuilder{
			method: method,
			route:  fmt.Sprintf(routeFormat, routeFields...),
			body:   bodyBytes,
		}.Test(t)

		err = json.Unmarshal(respBytes, &resp)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "json error, got json: %q", string(respBytes)))
		}

		if resp.V != want {
			t.Fatalf("Expect: %d, got %d", want, resp.V)
		}
	}

	testHTTP(nil, "GET", "getwithquery?%s=%d&%s=%d", "A", A, "B", B)
}

func TestGetWithRepeatedQueryClient(t *testing.T) {
	var req pb.GetWithRepeatedQueryRequest
	req.A = []int64{12, 45360}
	want := req.A[0] + req.A[1]

	svchttp, err := httpclient.New(httpserver.URL)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.GetWithRepeatedQuery(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != want {
		t.Fatalf("Expect: %d, got %d", want, resp.V)
	}
}

func TestGetWithRepeatedQueryRequest(t *testing.T) {
	var resp pb.GetWithRepeatedQueryResponse

	var A []int64
	A = []int64{12, 45360}
	want := A[0] + A[1]

	testHTTP := func(bodyBytes []byte, method, routeFormat string, routeFields ...interface{}) {
		respBytes, err := httpRequestBuilder{
			method: method,
			route:  fmt.Sprintf(routeFormat, routeFields...),
			body:   bodyBytes,
		}.Test(t)

		err = json.Unmarshal(respBytes, &resp)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "json error, got json: %q", string(respBytes)))
		}

		if resp.V != want {
			t.Fatalf("Expect: %d, got %d", want, resp.V)
		}
	}

	testHTTP(nil, "GET", "getwithrepeatedquery?%s=[%d,%d]", "A", A[0], A[1])
	// csv style
	//testHTTP(nil, "GET", "getwithrepeatedquery?%s=%d,%d", "A", A[0], A[1])
	// multi / golang style
	//testHTTP(nil, "GET", "getwithrepeatedquery?%s=%d&%s=%d]", "A", A[0], "A", A[1])
}

func TestPostWithNestedMessageBodyClient(t *testing.T) {
	var req pb.PostWithNestedMessageBodyRequest
	var reqNM pb.NestedMessage

	reqNM.A = 12
	reqNM.B = 45360
	req.NM = &reqNM
	want := req.NM.A + req.NM.B

	svchttp, err := httpclient.New(httpserver.URL)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.PostWithNestedMessageBody(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != want {
		t.Fatalf("Expect: %d, got %d", want, resp.V)
	}
}

func TestPostWithNestedMessageBodyRequest(t *testing.T) {
	var resp pb.PostWithNestedMessageBodyResponse

	var A, B int64
	A = 12
	B = 45360
	want := A + B

	testHTTP := func(bodyBytes []byte, method, routeFormat string, routeFields ...interface{}) {
		respBytes, err := httpRequestBuilder{
			method: method,
			route:  fmt.Sprintf(routeFormat, routeFields...),
			body:   bodyBytes,
		}.Test(t)

		err = json.Unmarshal(respBytes, &resp)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "json error, got json: %q", string(respBytes)))
		}

		if resp.V != want {
			t.Fatalf("Expect: %d, got %d", want, resp.V)
		}
	}

	jsonStr := fmt.Sprintf(`{ "NM": { "A": %d, "B": %d}}`, A, B)

	testHTTP([]byte(jsonStr), "POST", "postwithnestedmessagebody")
}

type httpRequestBuilder struct {
	method string
	route  string
	body   []byte
}

func (h httpRequestBuilder) Test(t *testing.T) ([]byte, error) {
	t.Logf("Method: %q | Route: %q", h.method, h.route)
	httpReq, err := http.NewRequest(h.method, httpserver.URL+"/"+h.route, bytes.NewReader(h.body))
	if err != nil {
		return nil, err
	}

	return testHTTPRequest(httpReq)
}

func testHTTPRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not end http request")
	}
	defer httpResp.Body.Close()

	respBytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read http body")
	}

	return respBytes, nil
}
