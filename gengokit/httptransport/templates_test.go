package httptransport

import (
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func TestGenClientEncode(t *testing.T) {
	binding := &Binding{
		Label:        "SumZero",
		PathTemplate: "/sum/{a}",
		BasePath:     "/sum/",
		Verb:         "get",
		Fields: []*Field{
			&Field{
				Name:          "a",
				CamelName:     "A",
				LowCamelName:  "a",
				LocalName:     "ASum",
				Location:      "path",
				ProtobufType:  "TYPE_INT64",
				GoType:        "int64",
				ProtobufLabel: "LABEL_OPTIONAL",
				ConvertFunc:   "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				IsBaseType:    true,
			},
			&Field{
				Name:          "b",
				CamelName:     "B",
				LowCamelName:  "b",
				LocalName:     "BSum",
				Location:      "query",
				ProtobufType:  "TYPE_INT64",
				GoType:        "int64",
				ProtobufLabel: "LABEL_OPTIONAL",
				ConvertFunc:   "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				IsBaseType:    true,
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
	req := request.(*pb.SumRequest)
	_ = req

	// Set the path parameters
	path := strings.Join([]string{
		"",
		"sum",
		fmt.Sprint(req.A),
	}, "/")
	//r.URL.Scheme,
	//r.URL.Host,
	u, err := url.Parse(path)
	if err != nil {
		return err
	}
	r.URL.RawPath = u.RawPath
	r.URL.Path = u.Path

	// Set the query parameters
	values := r.URL.Query()
	values.Add("b", fmt.Sprint(req.B))

	r.URL.RawQuery = values.Encode()

	// Set the body parameters
	var buf bytes.Buffer
	toRet := map[string]interface{}{}
	if err := json.NewEncoder(&buf).Encode(toRet); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	fmt.Printf("URL: %v\n", r.URL)
	return nil
}

`
	if got, want := str, desired; got != want {
		t.Errorf("Generated code differs from result.\ngot = %s\nwant = %s", got, want)
		t.Log(DiffStrings(got, want))
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
				Name:           "a",
				CamelName:      "A",
				LowCamelName:   "a",
				LocalName:      "ASum",
				Location:       "path",
				ProtobufType:   "TYPE_INT64",
				GoType:         "int64",
				ProtobufLabel:  "LABEL_OPTIONAL",
				ConvertFunc:    "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				TypeConversion: "ASum",
				IsBaseType:     true,
			},
			&Field{
				Name:           "b",
				CamelName:      "B",
				LowCamelName:   "b",
				LocalName:      "BSum",
				Location:       "query",
				ProtobufType:   "TYPE_INT64",
				GoType:         "int64",
				ProtobufLabel:  "LABEL_OPTIONAL",
				ConvertFunc:    "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				TypeConversion: "BSum",
				IsBaseType:     true,
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

	pathParams, err := PathParams(r.URL.Path, "/sum/{a}")
	_ = pathParams
	// TODO: Better error handling
	if err != nil {
		fmt.Printf("Error while reading path params: %v\n", err)
		return nil, err
	}
	queryParams, err := QueryParams(r.URL.Query())
	_ = queryParams
	// TODO: Better error handling
	if err != nil {
		fmt.Printf("Error while reading query params: %v\n", err)
		return nil, err
	}

	ASumStr := pathParams["a"]
	ASum, err := strconv.ParseInt(ASumStr, 10, 64)
	// TODO: Better error handling
	if err != nil {
		fmt.Printf("Error while extracting ASum from path: %v\n", err)
		fmt.Printf("pathParams: %v\n", pathParams)
		return nil, err
	}
	req.A = ASum

	BSumStr := queryParams["b"]
	BSum, err := strconv.ParseInt(BSumStr, 10, 64)
	// TODO: Better error handling
	if err != nil {
		fmt.Printf("Error while extracting BSum from query: %v\n", err)
		fmt.Printf("queryParams: %v\n", queryParams)
		return nil, err
	}
	req.B = BSum

	return &req, err
}

`
	if got, want := str, desired; got != want {
		t.Errorf("Generated code differs from result.\ngot = %s\nwant = %s", got, want)
		t.Log(DiffStrings(got, want))
	}
}

func TestHTTPAssistFuncs(t *testing.T) {
	tmplfncs := FormatCode(HTTPAssistFuncs)
	source, err := AllFuncSourceCode(BuildParamMap)
	if err != nil {
		t.Fatalf("Couldn't get source code of functions: %v", err)
	}

	if got, want := tmplfncs, FormatCode(source); got != want {
		t.Errorf("Assistant functions in templates differ from the source of those functions as they exist within the codebase")
		t.Log(DiffStrings(got, want))
	}
}

func DiffStrings(a, b string) string {
	t := difflib.UnifiedDiff{
		A:        difflib.SplitLines(a),
		B:        difflib.SplitLines(b),
		FromFile: "A",
		ToFile:   "B",
		Context:  5,
	}
	text, _ := difflib.GetUnifiedDiffString(t)
	return text
}
