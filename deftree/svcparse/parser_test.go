package svcparse

import (
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
)

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

func TestUnderscoreIdent(t *testing.T) {
	r := strings.NewReader("service Example_Service {}")
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}
	if len(svc.Methods) != 0 {
		t.Errorf("Parser found too many methods, expected 0, got %v\n", len(svc.Methods))
	}
}

func TestTrailingCommentsThreeDeep(t *testing.T) {
	r := strings.NewReader(`
service Example_Service {
	rpc Example(Empty) returns (Empty) {
		option (google.api.http) = {
			// Some example comment
			get: "/ExampleGet"
			body: "*"

			additional_bindings {
				post: "/ExamplePost"
			}
			// Testing comments
		};
	}
}
`)
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}

	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}

	if got, want := svc.Name, "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.Name, "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType, "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType, "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HTTPBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "get",
					Description: "// Some example comment\n",
					Kind:        "get",
					Value:       "/ExampleGet",
				},
				&Field{
					Name:  "body",
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	if got, want := meth.HTTPBindings, bindings; !reflect.DeepEqual(got, want) {
		t.Log(DiffStrings(spew.Sdump(got), spew.Sdump(want)))
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}
}

func TestTrailingCommentsTwoDeep(t *testing.T) {
	r := strings.NewReader(`
service Example_Service {
	rpc Example(Empty) returns (Empty) {
		option (google.api.http) = {
			// Some example comment
			get: "/ExampleGet"
			body: "*"

			additional_bindings {
				post: "/ExamplePost"
			}
		};
		// Testing comments
	}
}
`)
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}

	if got, want := svc.Name, "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.Name, "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType, "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType, "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HTTPBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "get",
					Description: "// Some example comment\n",
					Kind:        "get",
					Value:       "/ExampleGet",
				},
				&Field{
					Name:  "body",
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}

	if got, want := meth.HTTPBindings, bindings; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}
}

func TestTrailingCommentsOneDeep(t *testing.T) {
	r := strings.NewReader(`
service Example_Service {
	rpc Example(Empty) returns (Empty) {
		option (google.api.http) = {
			// Some example comment
			get: "/ExampleGet"
			body: "*"

			additional_bindings {
				post: "/ExamplePost"
			}
		};
	}
	// Testing comments
}
`)
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}

	if got, want := svc.Name, "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.Name, "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType, "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType, "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HTTPBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "get",
					Description: "// Some example comment\n",
					Kind:        "get",
					Value:       "/ExampleGet",
				},
				&Field{
					Name:  "body",
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}

	if got, want := meth.HTTPBindings, bindings; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}
}

func TestMultipleRpc(t *testing.T) {
	r := strings.NewReader(`
service Example_Service {
	rpc Example(Empty) returns (Empty) {
		option (google.api.http) = {
			// Some example comment
			get: "/ExampleGet"
			body: "*"

			additional_bindings {
				post: "/ExamplePost"
			}
		};
	}
	rpc SecondExample(Empty) returns (Empty) {
		option (google.api.http) = {
			// Second group of example comments
			get: "/SecondExampleGet"
			body: "*"

			// Second group of additional bindings
			additional_bindings {
				// Second binding, this time for post
				post: "/ExamplePost"
			}
		};
	}
}
`)
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}

	if got, want := svc.Name, "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 2; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	methone := svc.Methods[0]
	if got, want := methone.Name, "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType, "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType, "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	methtwo := svc.Methods[1]
	if got, want := methtwo.Name, "SecondExample"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType, "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType, "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(methone.HTTPBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindingsone := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "get",
					Description: "// Some example comment\n",
					Kind:        "get",
					Value:       "/ExampleGet",
				},
				&Field{
					Name:  "body",
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}

	if got, want := methone.HTTPBindings, bindingsone; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingstwo := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "post",
					Description: "// Second binding, this time for post\n",
					Kind:        "post",
					Value:       "/ExamplePost",
				},
			},
		},
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:        "get",
					Description: "// Second group of example comments\n",
					Kind:        "get",
					Value:       "/SecondExampleGet",
				},
				&Field{
					Name:  "body",
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}

	if got, want := methtwo.HTTPBindings, bindingstwo; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}
}

func TestMultipleRpcWithStream(t *testing.T) {
	r := strings.NewReader(`
service FlowCombination {
	rpc RpcEmptyStream(EmptyProto) returns (stream EmptyProto) {
		option (google.api.http) = {
			post: "/rpc/empty/stream"
		};
	}
	rpc StreamEmptyRpc(stream EmptyProto) returns (EmptyProto) {
		option (google.api.http) = {
			post: "/stream/empty/rpc"
		};
	}
	rpc StreamEmptyStream(stream EmptyProto) returns (stream EmptyProto) {
		option (google.api.http) = {
			post: "/stream/empty/stream"
		};
	}
}
`)
	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}

	if got, want := svc.Name, "FlowCombination"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 3; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}

	methone := svc.Methods[0]
	if got, want := methone.Name, "RpcEmptyStream"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType, "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType, "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	methtwo := svc.Methods[1]
	if got, want := methtwo.Name, "StreamEmptyRpc"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methtwo.RequestType, "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methtwo.ResponseType, "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	meththree := svc.Methods[2]
	if got, want := meththree.Name, "StreamEmptyStream"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meththree.RequestType, "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meththree.ResponseType, "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	bindingsone := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/rpc/empty/stream",
				},
			},
		},
	}

	if got, want := methone.HTTPBindings, bindingsone; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingstwo := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/stream/empty/rpc",
				},
			},
		},
	}
	if got, want := methtwo.HTTPBindings, bindingstwo; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingsthree := []*HTTPBinding{
		&HTTPBinding{
			Fields: []*Field{
				&Field{
					Name:  "post",
					Kind:  "post",
					Value: "/stream/empty/stream",
				},
			},
		},
	}

	if got, want := meththree.HTTPBindings, bindingsthree; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}
}

func TestOddComments(t *testing.T) {
	r := strings.NewReader(`
service FlowCombination {
	/* lots */
	/* of */
	/* comments */
	rpc RpcEmptyStream(EmptyProto) returns (stream EmptyProto) {
	/* lots */
	/* of */
	/* comments */
		option (google.api.http) = {
			post: "/rpc/empty/stream"
		};
	}
	/* lots */ /* of */ /* comments */
	rpc StreamEmptyRpc(stream EmptyProto) returns (EmptyProto) {
	/* lots */ /* of */ /* comments */
		option (google.api.http) = {
	/* lots */ /* of */ /* comments */
			post: "/stream/empty/rpc"
	/* lots */ /* of */ /* comments */
		};
	/* lots */ /* of */ /* comments */
	}
	rpc StreamEmptyStream(stream EmptyProto) returns (stream EmptyProto) {
		option (google.api.http) = {
			post: "/stream/empty/stream"
	/* lots */ /* of */ /* comments */
		};
	/* lots */ /* of */ /* comments */
	}
	/* lots */ /* of */ /* comments */
}
	`)

	lex := NewSvcLexer(r)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}
}
