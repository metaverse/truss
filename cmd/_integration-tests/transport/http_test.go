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
	"context"
	// This Service
	pb "github.com/tuneinc/truss/cmd/_integration-tests/transport/proto"
	httpclient "github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/svc/client/http"

	"github.com/pkg/errors"
)

var httpAddr string
var nonJSONHTTPAddr string

const brokenHTTPResponse = `<html> Not json </html>`
const brokenHTTPRequest = brokenHTTPResponse

func TestGetWithQueryClient(t *testing.T) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	want := req.A + req.B

	svchttp, err := httpclient.New(httpAddr, httpclient.CtxValuesToSend("request-url", "transport"))
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
	testHTTP(t, &resp, &expects, nil, "GET", "getwithrepeatedquery?%s=%d,%d", "A", A[0], A[1])
	// multi / golang style
	//testHTTP(nil, "GET", "getwithrepeatedquery?%s=%d&%s=%d]", "A", A[0], "A", A[1])
}

func TestGetWithEnumQueryClient(t *testing.T) {
	var req pb.GetWithEnumQueryRequest
	req.In = pb.TestStatus_test_passed
	want := pb.TestStatus_test_passed

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.GetWithEnumQuery(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.Out != want {
		t.Fatalf("Expect: %d, got %d", want, resp.Out)
	}
}

func TestGetWithEnumQueryRequest(t *testing.T) {
	resp := pb.GetWithEnumQueryResponse{}

	expects := pb.GetWithEnumQueryResponse{
		Out: pb.TestStatus_test_passed,
	}

	testHTTP(t, &resp, &expects, nil, "GET", "getwithenumquery?in=%d", pb.TestStatus_test_passed)
	// csv style
	testHTTP(t, &resp, &expects, nil, "GET", "getwithenumquery?in=%d", pb.TestStatus_test_passed)
	// multi / golang style
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
		t.Fatalf("cannot create httpclient: %q", err)
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

func TestGetWithEnumPathClient(t *testing.T) {
	var req pb.GetWithEnumQueryRequest
	req.In = pb.TestStatus_test_passed
	want := pb.TestStatus_test_passed

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.GetWithEnumPath(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.Out != want {
		t.Fatalf("Expect: %d, got %d", want, resp.Out)
	}
}

func TestGetWithEnumPathRequest(t *testing.T) {
	resp := pb.GetWithEnumQueryResponse{}

	expects := pb.GetWithEnumQueryResponse{
		Out: pb.TestStatus_test_passed,
	}

	testHTTP(t, &resp, &expects, nil, "GET", "getwithenumpath/%d", pb.TestStatus_test_passed)
	// csv style
	testHTTP(t, &resp, &expects, nil, "GET", "getwithenumpath/%d", pb.TestStatus_test_passed)
	// multi / golang style
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

// Certain RPC's with strange names (names which change when passed through the
// camelcase function) might break on HTTP encode/decode generation. Here we
// call an RPC with an one of those odd names which returns an empty message.
func TestStrangeRPCName(t *testing.T) {
	var req pb.Empty

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.X2AOddRPCName(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	var want *pb.Empty
	want = &pb.Empty{}
	if !reflect.DeepEqual(*resp, *want) {
		t.Fatalf("Expect: %d, got %d", want, resp)
	}
}

// Test that if a truss client receives a non-json response from a "truss"
// server, that we put that response body in the error message. To allow for
// developers to see the request body in the errors.
func TestNonJSONResponseBodyFromClientCallIsInError(t *testing.T) {
	svchttp, err := httpclient.New(nonJSONHTTPAddr)
	if err != nil {
		t.Fatalf("cannot create httpclient: %q", err)
	}

	var req pb.Empty
	_, err = svchttp.ErrorRPCNonJSON(context.Background(), &req)
	if err == nil {
		t.Fatal("Expected error from non-json response with http client")
	}

	if !strings.Contains(err.Error(), brokenHTTPResponse) {
		t.Fatalf("Expected error to contain `%s`; error is `%s`", brokenHTTPResponse, err.Error())
	}
}

// Test that if a non json response is recieved that is greater than 8KB, that
// we only put the first 8KB error, as to not flood memory with huge errors.
func TestNonJSONResponseBodyFromClientCallIsLessThan8KB(t *testing.T) {
	svchttp, err := httpclient.New(nonJSONHTTPAddr)
	if err != nil {
		t.Fatalf("cannot create httpclient: %q", err)
	}

	var req pb.Empty
	_, err = svchttp.ErrorRPCNonJSONLong(context.Background(), &req)
	if err == nil {
		t.Fatal("Expected error from non-json response with http client")
	}

	l := len(err.Error())
	// Add 200 for padding for the actual error message in addition to the response body
	if l > 8196+200 {
		t.Fatalf("Expected error to be less than 8KB with a little padding, actual %d", l)
	}
	t.Log("Non JSON response length", l)
}

// Test that if a truss server receives a non-json request, we put that request
// body in the error message. To allow for developers to see the request body in the errors.
func TestNonJSONRequestBodyIsInError(t *testing.T) {
	// Put some bad data into the body
	req, err := http.NewRequest("POST", httpAddr+"/error/non/json", strings.NewReader(brokenHTTPRequest))
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	respBytes, err := testHTTPRequest(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}

	var resp map[string]string
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatalf("cannot unmarshal bytes: %s", respBytes)
	}

	if !strings.Contains(resp["error"], brokenHTTPRequest) {
		t.Fatalf("Expected error to contain `%s`; error is `%s`", brokenHTTPResponse, resp["error"])
	}
}

// Test that if a non json request is received that is greater than 8KB, that
// we only put the first 8KB error, as to not flood memory with huge errors.
func TestNonJSONRequestBodyIsLessThan8KB(t *testing.T) {
	// Put a 16kb of bad data into the body
	badBody := make([]byte, 8196*2)
	for i := 0; i < 8196*2; i++ {
		badBody = append(badBody, byte(i%256))
	}
	req, err := http.NewRequest("POST", httpAddr+"/error/non/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	respBytes, err := testHTTPRequest(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}

	var resp map[string]string
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatalf("cannot unmarshal bytes: %s", respBytes)
	}

	l := len(resp["error"])
	// Add 200 for padding for the actual error message in addition to the response body
	if l > 8196+200 {
		t.Fatalf("Expected error to be less than 8KB with a little padding, actual %d", l)
	}
	t.Log("Non JSON request length", l)
}

func TestResponseContentType(t *testing.T) {
	req, err := http.NewRequest("GET", httpAddr+"/content/type", nil)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}
	defer httpResp.Body.Close()

	got, want := httpResp.Header.Get("Content-Type"), "application/json"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("Expected content type to have `%s` got `%s`", want, got)
	}
}

func TestHTTPErrorStatusCodeAndNilHeaders(t *testing.T) {
	// See handlers/handlers.go for implementation
	// Returns status code http.StatusTeapot
	req, err := http.NewRequest("GET", httpAddr+"/status/code", nil)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make end http request"))
	}
	defer httpResp.Body.Close()

	got, want := httpResp.StatusCode, http.StatusTeapot
	if got != want {
		t.Fatalf("Expected status code:`%d`, Got status code: `%d`", want, got)
	}
}

func TestHTTPErrorStatusCodeAndHeaders(t *testing.T) {
	// See handlers/handlers.go for implementation
	// Returns status code http.StatusTeapot and headers
	// Foo: Bar
	// Test: A, B

	req, err := http.NewRequest("GET", httpAddr+"/status/code/and/headers", nil)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}
	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make end http request"))
	}
	defer httpResp.Body.Close()

	got, want := httpResp.Header, map[string][]string{
		"Foo":  []string{"Bar"},
		"Test": []string{"A", "B"},
	}

	for k := range want {
		_, ok := got[k]
		if !ok {
			t.Fatalf("Expected header `%s:%s`; Got no header with key: `%s`", k, want[k], k)
		}

		for i := range got[k] {
			if got[k][i] != want[k][i] {
				t.Fatalf("Expected Header `%s:%s`; Got `%s:%s`", k, want[k], k, got[k])
			}
		}
	}

	gotStatus, wantStatus := httpResp.StatusCode, http.StatusTeapot
	if gotStatus != wantStatus {
		t.Fatalf("Expected status code:`%d`, Got status code: `%d`", wantStatus, gotStatus)
	}
}

// Test that if a truss server receives a non-json request, that the status code is 400 http.StatusBadRequest
// body in the error message. To allow for developers to see the request body in the errors.
func TestNonJSONRequestBodyReturnsResponseWithStatusCode400(t *testing.T) {
	// Put some bad data into the body
	req, err := http.NewRequest("POST", httpAddr+"/error/non/json", strings.NewReader(brokenHTTPRequest))
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot construct http request"))
	}

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatal(errors.Wrap(err, "cannot make http request"))
	}
	defer httpResp.Body.Close()

	got, want := httpResp.StatusCode, http.StatusBadRequest
	if got != want {
		t.Fatalf("Expected status code:`%d`, Got status code: `%d`", want, got)
	}
}

// Manually test that we can make HTTP requests with a custom verb
func TestCustomVerbRequest(t *testing.T) {
	var resp pb.GetWithQueryResponse

	var A, B int64
	A = 12
	B = 45360
	expects := pb.GetWithQueryResponse{
		V: A + B,
	}

	testHTTP(t, &resp, &expects, nil, "CUSTOMVERB", "customverb?%s=%d&%s=%d", "A", A, "B", B)
}

// Test that we can use the generated client to make requests to methods which
// use custom verbs.
func TestCustomVerbClient(t *testing.T) {
	var req pb.GetWithQueryRequest
	req.A = 12
	req.B = 45360
	want := req.A + req.B

	svchttp, err := httpclient.New(httpAddr)
	if err != nil {
		t.Fatalf("failed to create httpclient: %q", err)
	}

	resp, err := svchttp.CustomVerb(context.Background(), &req)
	if err != nil {
		t.Fatalf("httpclient returned error: %q", err)
	}

	if resp.V != want {
		t.Fatalf("Expect: %d, got %d", want, resp.V)
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
	resp,
	expects interface{},
	bodyBytes []byte,
	method,
	routeFormat string,
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

	// These are set to allow TestGetWithQueryRequest to pass since it will
	// use the same handler as the client version
	httpReq.Header.Set("request-url", httpReq.URL.Path)
	httpReq.Header.Set("transport", "HTTPJSON")

	return testHTTPRequest(httpReq)
}

func testHTTPRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot make http request")
	}
	defer httpResp.Body.Close()

	respBytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read http body")
	}

	return respBytes, nil
}
