package httptransport

// ServerDecodeTemplate is the template for generating the server-side decoding
// function for a particular Binding.
var ServerDecodeTemplate = `
{{ with $binding := .}}
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
{{end}}
`

// ClientEncodeTemplate is the template for generating the client-side encoding
// function for a particular Binding.
var ClientEncodeTemplate = `
{{ with $binding := .}}
// EncodeHTTP{{$binding.Label}}Request is a transport/http.EncodeRequestFunc
// that encodes a {{ToLower $binding.Parent.Name}} request into the various portions of
// the http request (path, query, and body).
func EncodeHTTP{{$binding.Label}}Request(_ context.Context, r *http.Request, request interface{}) error {
	fmt.Printf("Encoding request %v\n", request)
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
{{/*
	{{- if eq $field.ProtobufLabel "LABEL_REPEATED"}}
		for _, v := range req.{{$field.CamelName}} {
			values.Add("{{$field.Name}}", fmt.Sprint(v))
		}
*/}}
	{{if or (not $field.IsBaseType) $field.Repeated}}
		tmp, err = json.Marshal(req.{{$field.CamelName}})
		if err != nil {
			return errors.Wrap(err, "failed to marshal req.{{$field.CamelName}}")
		}
		values.Add("{{$field.Name}}", string(tmp))
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
{{end}}
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
// structs, and is used within the generated code of each microservice.
var HTTPAssistFuncs = PathParamsTemplate + BuildParamMapTemplate + RemoveBracesTemplate + QueryParamsTemplate
