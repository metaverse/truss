package test

import (
	"reflect"
	"testing"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	// 3d Party
	"golang.org/x/net/context"
	// This Service
	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transportpermutations-service"
	httpclient "github.com/TuneLab/go-truss/cmd/_integration-tests/transport/transportpermutations-service/generated/client/http"

	"github.com/pkg/errors"
)

var httpAddr string

func TestGetWithQueryClient(t *testing.T) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	want := req.A + req.B

	svchttp, err := httpclient.New(httpAddr)
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

// A manually-constructed HTTP request test, ensuring that a get with query
// parameters works even outside the behvior of the client.
func TestGetWithQueryRequest(t *testing.T) {
	var resp pb.GetWithQueryResponse

	var A, B int64
	A = 12
	B = 45360
	expects := pb.GetWithQueryResponse{
		V: A + B,
	}

	testHTTP(t, &resp, &expects, nil, "GET", "getwithquery?%s=%d&%s=%d", "A", A, "B", B)
}

func TestGetWithRepeatedQueryClient(t *testing.T) {
	var req pb.GetWithRepeatedQueryRequest
	req.A = []int64{12, 45360}
	want := req.A[0] + req.A[1]

	svchttp, err := httpclient.New(httpAddr)
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
	resp := pb.GetWithRepeatedQueryResponse{}

	var A []int64
	A = []int64{12, 45360}
	expects := pb.GetWithRepeatedQueryResponse{
		V: A[0] + A[1],
	}

	testHTTP(t, &resp, &expects, nil, "GET", "getwithrepeatedquery?%s=[%d,%d]", "A", A[0], A[1])
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

	svchttp, err := httpclient.New(httpAddr)
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
	expects := pb.PostWithNestedMessageBodyResponse{
		V: A + B,
	}
	jsonStr := fmt.Sprintf(`{ "NM": { "A": %d, "B": %d}}`, A, B)

	testHTTP(t, &resp, &expects, []byte(jsonStr), "POST", "postwithnestedmessagebody")
}

func TestCtxToCtxViaHTTPHeaderClient(t *testing.T) {
	var req pb.MetaRequest
	var key, value = "Truss-Auth-Header", "SECRET"
	req.Key = key

	// Create a new client telling it to send "Truss-Auth-Header" as a header
	svchttp, err := httpclient.New(httpAddr,
		httpclient.CtxValuesToSend(key))
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	// Create a context with the header key and value
	ctx := context.WithValue(context.Background(), key, value)

	// send the context
	resp, err := svchttp.CtxToCtx(ctx, &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != value {
		t.Fatalf("Expect: %q, got %q", value, resp.V)
	}
}

// Test different kinds of message field names from protobuf definition.
// e.g. "CamelCase", "snake_case", "__why_so_many_underscores"
func TestEchoOddNamesClient(t *testing.T) {
	req := pb.OddFieldNames{
		CamelCase:                12,
		SnakeCase:                24,
		XWhy_So_Many_Underscores: 36,
	}

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.EchoOddNames(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}
	if !reflect.DeepEqual(resp, &req) {
		t.Fatalf("Expected req and resp to be identical, instead: \n%+v\n%+v", req, *resp)
	}
}

func TestCtxToCtxViaHTTPHeaderRequest(t *testing.T) {
	var resp pb.MetaResponse
	var key, value = "Truss-Auth-Header", "SECRET"

	jsonStr := fmt.Sprintf(`{ "Key": %q }`, key)
	fmt.Println(jsonStr)

	req, err := http.NewRequest("POST", httpAddr+"/"+"ctxtoctx", strings.NewReader(jsonStr))
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	req.Header.Set(key, value)

	respBytes, err := testHTTPRequest(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}

	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "json error, got response: %q", string(respBytes)))
	}

	if resp.V != value {
		t.Fatalf("Expect: %q, got %q", value, resp.V)
	}
}

// Test that making a get request through the client with capital letters in
// the path functions correctly.
func TestGetWithCapsPathClient(t *testing.T) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	want := req.A + req.B

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.GetWithCapsPath(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != want {
		t.Fatalf("Expect: %d, got %d", want, resp.V)
	}
}

// A manually created request verifying the server handles paths with capital
// letters.
func TestGetWithCapsPathRequest(t *testing.T) {
	var resp pb.GetWithQueryResponse

	var A, B int64
	A = 12
	B = 45360
	expects := pb.GetWithQueryResponse{
		V: A + B,
	}

	testHTTP(t, &resp, &expects, nil, "GET", "get/With/CapsPath?%s=%d&%s=%d", "A", A, "B", B)
}

// Test that we can manually insert parameters into the path and recieve a
// correct response.
func TestGetWithPathParams(t *testing.T) {
	var resp pb.GetWithQueryResponse
	var A, B int64
	A = 12
	B = 45360
	expects := pb.GetWithQueryResponse{
		V: A + B,
	}

	testHTTP(t, &resp, &expects, nil, "GET", "path/%d/%d", A, B)
}

// A manually created request verifying that the server properly responds with
// an error if a request is made with incomplete path parameters.
func TestGetWithPathParamsRequest_IncompletePath(t *testing.T) {
	var A int64
	A = 12
	path := fmt.Sprintf("/path/%d/", A)

	httpReq, err := http.NewRequest("GET", httpAddr+path, strings.NewReader(""))
	if err != nil {
		t.Errorf("couldn't create request", err)
	}
	respBytes, err := testHTTPRequest(httpReq)

	var resp map[string]string
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatalf("couldn't unmarshal bytes: %v", err)
	}

	want := map[string]string{
		"error": "couldn't unmarshal path parameters: Expected a path containing 4 parts, provided path contains 3 parts",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Fatalf("Expect: %v, got %v", want, resp)
	}
}

func TestErrorRPCReturnsJSONError(t *testing.T) {
	req, err := http.NewRequest("GET", httpAddr+"/"+"error", strings.NewReader(""))
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	respBytes, err := testHTTPRequest(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}

	jsonOut := make(map[string]interface{})
	err = json.Unmarshal(respBytes, &jsonOut)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "json error, got response: %q", string(respBytes)))
	}

	if jsonOut["error"] == nil {
		t.Fatal("http transport did not send error as json")
	}
}

// Helpers

// Generic way to test that making an HTTP request returns the expected data,
// where the response is a JSON object of some kind.
//
// The resp parameter must be a pointer to an uninitialized struct of some
// kind, while the expects parameter must be of the same struct type as resp,
// but is initialized with the data that should be returned from the HTTP
// request. Note as well: Due to quirks in how json.Unmarshal works, both resp
// and expects must be pointers to structs.
func testHTTP(
	t *testing.T,
	resp, expects interface{},
	bodyBytes []byte,
	method, routeFormat string,
	routeFields ...interface{}) {
	respBytes, err := httpRequestBuilder{
		method: method,
		route:  fmt.Sprintf(routeFormat, routeFields...),
		body:   bodyBytes,
	}.Test(t)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}

	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "json error, got response: %q", string(respBytes)))
	}

	if !reflect.DeepEqual(resp, expects) {
		t.Fatalf("Expect: %+v, got %+v", expects, resp)
	}
}

type httpRequestBuilder struct {
	method string
	route  string
	body   []byte
}

func (h httpRequestBuilder) Test(t *testing.T) ([]byte, error) {
	t.Logf("Method: %q | Route: %q", h.method, h.route)
	httpReq, err := http.NewRequest(h.method, httpAddr+"/"+h.route, bytes.NewReader(h.body))
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
