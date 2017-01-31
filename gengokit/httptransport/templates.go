package httptransport

// ServerDecodeTemplate is the template for generating the server-side decoding
// function for a particular Binding.
var ServerDecodeTemplate = `
{{- with $binding := . -}}
	// DecodeHTTP{{$binding.Label}}Request is a transport/http.DecodeRequestFunc that
	// decodes a JSON-encoded {{ToLower $binding.Parent.Name}} request from the HTTP request
	// body. Primarily useful in a server.
	func DecodeHTTP{{$binding.Label}}Request(_ context.Context, r *http.Request) (interface{}, error) {
		var req pb.{{GoName $binding.Parent.RequestType}}
		err := json.NewDecoder(r.Body).Decode(&req)
		// err = io.EOF if r.Body was empty
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "decoding body of http request")
		}

		pathParams, err := PathParams(r.URL.Path, "{{$binding.PathTemplate}}")
		_ = pathParams
		if err != nil {
			fmt.Printf("Error while reading path params: %v\n", err)
			return nil, errors.Wrap(err, "couldn't unmarshal path parameters")
		}
		queryParams, err := QueryParams(r.URL.Query())
		_ = queryParams
		if err != nil {
			fmt.Printf("Error while reading query params: %v\n", err)
			return nil, errors.Wrapf(err, "Error while reading query params: %v", r.URL.Query())
		}
	{{range $field := $binding.Fields}}
		{{if ne $field.Location "body"}}
			{{$field.GenQueryUnmarshaler}}
		{{end}}
	{{end}}
		return &req, err
	}
{{- end -}}
`

// ClientEncodeTemplate is the template for generating the client-side encoding
// function for a particular Binding.
var ClientEncodeTemplate = `
{{- with $binding := . -}}
	// EncodeHTTP{{$binding.Label}}Request is a transport/http.EncodeRequestFunc
	// that encodes a {{ToLower $binding.Parent.Name}} request into the various portions of
	// the http request (path, query, and body).
	func EncodeHTTP{{$binding.Label}}Request(_ context.Context, r *http.Request, request interface{}) error {
		fmt.Printf("Encoding request %v\n", request)
		strval := ""
		_ = strval
		req := request.(*pb.{{GoName $binding.Parent.RequestType}})
		_ = req

		// Set the path parameters
		path := strings.Join([]string{
		{{- range $section := $binding.PathSections}}
			{{$section}},
		{{- end}}
		}, "/")
		u, err := url.Parse(path)
		if err != nil {
			return errors.Wrapf(err, "couldn't unmarshal path %q", path)
		}
		r.URL.RawPath = u.RawPath
		r.URL.Path = u.Path

		// Set the query parameters
		values := r.URL.Query()
		var tmp []byte
		_ = tmp
		{{- range $field := $binding.Fields }}
			{{- if eq $field.Location "query"}}
				{{if or (not $field.IsBaseType) $field.Repeated}}
					tmp, err = json.Marshal(req.{{$field.CamelName}})
					if err != nil {
						return errors.Wrap(err, "failed to marshal req.{{$field.CamelName}}")
					}
					strval = string(tmp)
					values.Add("{{$field.Name}}", strval)
				{{else}}
					values.Add("{{$field.Name}}", fmt.Sprint(req.{{$field.CamelName}}))
				{{- end }}
			{{- end }}
		{{- end}}

		r.URL.RawQuery = values.Encode()

		// Set the body parameters
		var buf bytes.Buffer
		toRet := map[string]interface{}{
		{{- range $field := $binding.Fields -}}
			{{if eq $field.Location "body"}}
				"{{$field.CamelName}}" : req.{{$field.CamelName}},
			{{end}}
		{{- end -}}
		}
		if err := json.NewEncoder(&buf).Encode(toRet); err != nil {
			return errors.Wrapf(err, "couldn't encode body as json %v", toRet)
		}
		r.Body = ioutil.NopCloser(&buf)
		fmt.Printf("URL: %v\n", r.URL)
		return nil
	}
{{- end -}}
`

// WARNING: Changing the contents of these strings, even a little bit, will cause tests
// to fail. So don't change them purely because you think they look a little
// funny.

// PathParamsTemplate is a source code literal of the code for the PathParams
// function found in embeddeable_funcs.go
var PathParamsTemplate = `// PathParams takes a url and a gRPC-annotation style url template, and
// returns a map of the named parameters in the template and their values in
// the given url.
//
// PathParams does not support the entirety of the URL template syntax defined
// in third_party/googleapis/google/api/httprule.proto. Only a small subset of
// the functionality defined there is implemented here.
func PathParams(url string, urlTmpl string) (map[string]string, error) {
	rv := map[string]string{}
	pmp := BuildParamMap(urlTmpl)

	expectedLen := len(strings.Split(strings.TrimRight(urlTmpl, "/"), "/"))
	recievedLen := len(strings.Split(strings.TrimRight(url, "/"), "/"))
	if expectedLen != recievedLen {
		return nil, fmt.Errorf("Expected a path containing %d parts, provided path contains %d parts", expectedLen, recievedLen)
	}

	parts := strings.Split(url, "/")
	for k, v := range pmp {
		rv[k] = parts[v]
	}

	return rv, nil
}`

// BuildParamMapTemplate is a source code literal of the code for the
// BuildParamMap function found in embeddeable_funcs.go
var BuildParamMapTemplate = `
// BuildParamMap takes a string representing a url template and returns a map
// indicating the location of each parameter within that url, where the
// location is the index as if in a slash-separated sequence of path
// components. For example, given the url template:
//
//     "/v1/{a}/{b}"
//
// The returned param map would look like:
//
//     map[string]int {
//         "a": 2,
//         "b": 3,
//     }
func BuildParamMap(urlTmpl string) map[string]int {
	rv := map[string]int{}

	parts := strings.Split(urlTmpl, "/")
	for idx, part := range parts {
		if strings.ContainsAny(part, "{}") {
			param := RemoveBraces(part)
			rv[param] = idx
		}
	}
	return rv
}`

// RemoveBracesTemplate is a source code literal of the code for the
// RemoveBraces function found in embeddeable_funcs.go
var RemoveBracesTemplate = `
// RemoveBraces replace all curly braces in the provided string, opening and
// closing, with empty strings.
func RemoveBraces(val string) string {
	val = strings.Replace(val, "{", "", -1)
	val = strings.Replace(val, "}", "", -1)
	return val
}`

// QueryParamsTemplate is a source code literal of the code for the QueryParams
// function found in embeddeable_funcs.go
var QueryParamsTemplate = `
// QueryParams takes query parameters in the form of url.Values, and returns a
// bare map of the string representation of each key to the string
// representation for each value. The representations of repeated query
// parameters is undefined.
func QueryParams(vals url.Values) (map[string]string, error) {

	rv := map[string]string{}
	for k, v := range vals {
		rv[k] = v[0]
	}
	return rv, nil
}
`

// HTTPAssistFuncs is a source code literal of all the helper functions used
// for encoding and decoding http request to and from generated protobuf
// structs, and is used within the generated code of each service.
var HTTPAssistFuncs = PathParamsTemplate + BuildParamMapTemplate + RemoveBracesTemplate + QueryParamsTemplate

var serverTemplate = `
package svc

// This file provides server-side bindings for the HTTP transport.
// It utilizes the transport/http.Server.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"io"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	httptransport "github.com/go-kit/kit/transport/http"

	// This service
	pb "{{.PBImportPath -}}"
)

var (
	_ = fmt.Sprint
	_ = bytes.Compare
	_ = strconv.Atoi
	_ = httptransport.NewServer
	_ = ioutil.NopCloser
	_ = pb.Register{{.Service.Name}}Server
	_ = io.Copy
)

// MakeHTTPHandler returns a handler that makes a set of endpoints available
// on predefined paths.
func MakeHTTPHandler(ctx context.Context, endpoints Endpoints, logger log.Logger) http.Handler {
	{{- if .HTTPHelper.Methods}}
		serverOptions := []httptransport.ServerOption{
			httptransport.ServerBefore(headersToContext),
			httptransport.ServerErrorEncoder(errorEncoder),
		}
	{{- end }}
	m := http.NewServeMux()

	{{range $method := .HTTPHelper.Methods}}
		{{range $binding := $method.Bindings}}
			m.Handle("{{$binding.BasePath}}", httptransport.NewServer(
				ctx,
				endpoints.{{$method.Name}}Endpoint,
				HttpDecodeLogger(DecodeHTTP{{$binding.Label}}Request, logger),
				EncodeHTTPGenericResponse,
				serverOptions...,
			))
		{{- end}}
	{{- end}}
	return m
}

func HttpDecodeLogger(next httptransport.DecodeRequestFunc, logger log.Logger) httptransport.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		logger.Log("method", r.Method, "url", r.URL.String())
		rv, err := next(ctx, r)
		if err != nil {
			logger.Log("method", r.Method, "url", r.URL.String(), "Error", err)
		}
		return rv, err
	}
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string ` + "`json:\"error\"`\n" + `
}

// Server Decode
{{range $method := .HTTPHelper.Methods}}
	{{range $binding := $method.Bindings}}
		{{$binding.GenServerDecode}}
	{{end}}
{{end}}

// Client Decode
{{range $method := .HTTPHelper.Methods}}
	// DecodeHTTP{{$method.Name}} is a transport/http.DecodeResponseFunc that decodes
	// a JSON-encoded {{GoName $method.ResponseType}} response from the HTTP response body.
	// If the response has a non-200 status code, we will interpret that as an
	// error and attempt to decode the specific error message from the response
	// body. Primarily useful in a client.
	func DecodeHTTP{{$method.Name}}Response(_ context.Context, r *http.Response) (interface{}, error) {
		if r.StatusCode != http.StatusOK {
			return nil, errorDecoder(r)
		}
		var resp pb.{{GoName $method.ResponseType}}
		err := json.NewDecoder(r.Body).Decode(&resp)
		return &resp, err
	}
{{end}}

// Client Encode
{{range $method := .HTTPHelper.Methods}}
	{{range $binding := $method.Bindings}}
		{{$binding.GenClientEncode}}
	{{end}}
{{end}}

// EncodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func EncodeHTTPGenericResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

// Helper functions

{{.HTTPHelper.PathParamsBuilder}}

func headersToContext(ctx context.Context, r *http.Request) context.Context {
	for k, _ := range r.Header {
		// The key is added both in http format (k) which has had
		// http.CanonicalHeaderKey called on it in transport as well as the
		// strings.ToLower which is the grpc metadata format of the key so
		// that it can be accessed in either format
		ctx = context.WithValue(ctx, k, r.Header.Get(k))
		ctx = context.WithValue(ctx, strings.ToLower(k), r.Header.Get(k))
	}

	return ctx
}
`

var clientTemplate = `
// Package http provides an HTTP client for the {{.Service.Name}} service.
package http

import (
	"net/url"
	"strings"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	// This Service
	svc "{{.ImportPath -}} /generated"
	pb "{{.PBImportPath -}}"
)

var (
	_ = endpoint.Chain
	_ = httptransport.NewClient
)

// New returns a service backed by an HTTP server living at the remote
// instance. We expect instance to come from a service discovery system, so
// likely of the form "host:port".
func New(instance string, options ...ClientOption) (pb.{{GoName .Service.Name}}Server, error) {
	var cc clientConfig

	for _, f := range options {
		err := f(&cc)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply option") }
	}

	{{ if .HTTPHelper.Methods }}
		clientOptions := []httptransport.ClientOption{
			httptransport.ClientBefore(
				contextValuesToHttpHeaders(cc.headers)),
		}
	{{ end }}

	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}
	_ = u

	{{range $method := .HTTPHelper.Methods}}
		{{range $binding := $method.Bindings}}
			var {{$binding.Label}}Endpoint endpoint.Endpoint
			{
				{{$binding.Label}}Endpoint = httptransport.NewClient(
					"{{$binding.Verb}}",
					copyURL(u, "{{$binding.BasePath}}"),
					svc.EncodeHTTP{{$binding.Label}}Request,
					svc.DecodeHTTP{{$method.Name}}Response,
					clientOptions...,
				).Endpoint()
			}
		{{- end}}
	{{- end}}

	return svc.Endpoints{
	{{range $method := .HTTPHelper.Methods -}}
		{{range $binding := $method.Bindings -}}
			{{$method.Name}}Endpoint:    {{$binding.Label}}Endpoint,
		{{end}}
	{{- end}}
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

type clientConfig struct {
	headers []string
}

// ClientOption is a function that modifies the client config
type ClientOption func(*clientConfig) error

// CtxValuesToSend configures the http client to pull the specified keys out of
// the context and add them to the http request as headers.  Note that keys
// will have net/http.CanonicalHeaderKey called on them before being send over
// the wire and that is the form they will be available in the server context.
func CtxValuesToSend(keys ...string) ClientOption {
	return func(o *clientConfig) error {
		o.headers = keys
		return nil
	}
}

func contextValuesToHttpHeaders(keys []string) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, k := range keys {
			if v, ok := ctx.Value(k).(string); ok {
				r.Header.Set(k, v)
			}
		}

		return ctx
	}
}
`
