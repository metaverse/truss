package httptransport

import (
	"strings"
	"testing"

	"github.com/TuneLab/go-truss/gengokit/gentesthelper"
	"github.com/TuneLab/go-truss/gengokit/httptransport/templates"
)

// Test that rendering certain templates will ouput the code we expect. The
// code we expect is either the source code literal defined in each test, or
// it's the source code of certain actual functions within this package (see
// embeddable-funcs.go for more info).

func TestGenClientEncode(t *testing.T) {
	binding := &Binding{
		Label:        "SumZero",
		PathTemplate: "/sum/{a}",
		BasePath:     "/sum/",
		Verb:         "get",
		Fields: []*Field{
			&Field{
				Name:         "a",
				CamelName:    "A",
				LowCamelName: "a",
				LocalName:    "ASum",
				Location:     "path",
				GoType:       "int64",
				ConvertFunc:  "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				IsBaseType:   true,
			},
			&Field{
				Name:         "b",
				CamelName:    "B",
				LowCamelName: "b",
				LocalName:    "BSum",
				Location:     "query",
				GoType:       "int64",
				ConvertFunc:  "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				IsBaseType:   true,
			},
		},
	}
	meth := &Method{
		Name:         "Sum",
		RequestType:  "SumRequest",
		ResponseType: "SumReply",
		Bindings: []*Binding{
			binding,
		},
	}
	binding.Parent = meth

	str, err := binding.GenClientEncode()
	if err != nil {
		t.Errorf("Failed to generate client code: %v", err)
	}
	desired := `

// EncodeHTTPSumZeroRequest is a transport/http.EncodeRequestFunc
// that encodes a sum request into the various portions of
// the http request (path, query, and body).
func EncodeHTTPSumZeroRequest(_ context.Context, r *http.Request, request interface{}) error {
	fmt.Printf("Encoding request %v\n", request)
	strval := ""
	_ = strval
	req := request.(*pb.SumRequest)
	_ = req

	// Set the path parameters
	path := strings.Join([]string{
		"",
		"sum",
		fmt.Sprint(req.A),
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

	values.Add("b", fmt.Sprint(req.B))

	r.URL.RawQuery = values.Encode()

	// Set the body parameters
	var buf bytes.Buffer
	toRet := request.(*pb.SumRequest)
	if err := json.NewEncoder(&buf).Encode(toRet); err != nil {
		return errors.Wrapf(err, "couldn't encode body as json %v", toRet)
	}
	r.Body = ioutil.NopCloser(&buf)
	fmt.Printf("URL: %v\n", r.URL)
	return nil
}

`
	if got, want := strings.TrimSpace(str), strings.TrimSpace(desired); got != want {
		t.Errorf("Generated code differs from result.\ngot = %s\nwant = %s", got, want)
		t.Log(gentesthelper.DiffStrings(got, want))
	}
}

func TestGenServerDecode(t *testing.T) {
	binding := &Binding{
		Label:        "SumZero",
		PathTemplate: "/sum/{a}",
		BasePath:     "/sum/",
		Verb:         "get",
		Fields: []*Field{
			&Field{
				Name:                       "a",
				QueryParamName:             "a",
				CamelName:                  "A",
				LowCamelName:               "a",
				LocalName:                  "ASum",
				Location:                   "path",
				GoType:                     "int64",
				ConvertFunc:                "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				ConvertFuncNeedsErrorCheck: true,
				TypeConversion:             "ASum",
				IsBaseType:                 true,
			},
			&Field{
				Name:                       "b",
				QueryParamName:             "b",
				CamelName:                  "B",
				LowCamelName:               "b",
				LocalName:                  "BSum",
				Location:                   "query",
				GoType:                     "int64",
				ConvertFunc:                "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				ConvertFuncNeedsErrorCheck: true,
				TypeConversion:             "BSum",
				IsBaseType:                 true,
			},
		},
	}
	meth := &Method{
		Name:         "Sum",
		RequestType:  "SumRequest",
		ResponseType: "SumReply",
		Bindings: []*Binding{
			binding,
		},
	}
	binding.Parent = meth

	str, err := binding.GenServerDecode()
	if err != nil {
		t.Errorf("Failed to generate server decode code: %v", err)
	}
	desired := `

// DecodeHTTPSumZeroRequest is a transport/http.DecodeRequestFunc that
// decodes a JSON-encoded sum request from the HTTP request
// body. Primarily useful in a server.
func DecodeHTTPSumZeroRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req pb.SumRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	// err = io.EOF if r.Body was empty
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "decoding body of http request")
	}

	pathParams, err := PathParams(r.URL.Path, "/sum/{a}")
	_ = pathParams
	if err != nil {
		return nil, errors.Wrap(err, "couldn't unmarshal path parameters")
	}

	queryParams := r.URL.Query()
	_ = queryParams

	ASumStr := pathParams["a"]
	ASum, err := strconv.ParseInt(ASumStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error while extracting ASum from path, pathParams: %v", pathParams))
	}
	req.A = ASum

	if BSumStrArr, ok := queryParams["b"]; ok {
		BSumStr := BSumStrArr[0]
		BSum, err := strconv.ParseInt(BSumStr, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Error while extracting BSum from query, queryParams: %v", queryParams))
		}
		req.B = BSum
	}

	return &req, err
}

`
	if got, want := strings.TrimSpace(str), strings.TrimSpace(desired); got != want {
		t.Errorf("Generated code differs from result.\ngot = %s\nwant = %s", got, want)
		t.Log(gentesthelper.DiffStrings(got, want))
	}
}

// Test that all the templated source code is identical to the source code
// found within the file 'embeddable_funcs.go'.
func TestHTTPAssistFuncs(t *testing.T) {
	tmplfncs := FormatCode(templates.HTTPAssistFuncs)
	// Get the source code for all the functions in the same source file as
	// the BuildParamMap function
	source, err := AllFuncSourceCode(BuildParamMap)
	if err != nil {
		t.Fatalf("Couldn't get source code of functions: %v", err)
	}

	if got, want := tmplfncs, FormatCode(source); got != want {
		t.Errorf("Assistant functions in templates differ from the source of those functions as they exist within the codebase")
		t.Log(gentesthelper.DiffStrings(got, want))
	}
}
