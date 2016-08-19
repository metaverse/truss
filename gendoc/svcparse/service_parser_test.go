package svcparse

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TuneLab/go-truss/gendoc/doctree"
)

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

	if got, want := svc.GetName(), "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.GetName(), "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType.GetName(), "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType.GetName(), "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HttpBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "get",
					Value: "/ExampleGet",
				},
				&doctree.BindingField{
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	// Have to use SetName/SetDescription methods after declaration since
	// `Name` and `Description are fields of a non-exported embedded struct
	bindings[0].Fields[0].SetName("post")
	bindings[1].Fields[0].SetName("get")
	bindings[1].Fields[0].SetDescription("Some example comment")
	bindings[1].Fields[1].SetName("body")

	if got, want := meth.HttpBindings, bindings; !reflect.DeepEqual(got, want) {
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

	if got, want := svc.GetName(), "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.GetName(), "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType.GetName(), "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType.GetName(), "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HttpBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "get",
					Value: "/ExampleGet",
				},
				&doctree.BindingField{
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	// Have to use SetName/SetDescription methods after declaration since
	// `Name` and `Description are fields of a non-exported embedded struct
	bindings[0].Fields[0].SetName("post")
	bindings[1].Fields[0].SetName("get")
	bindings[1].Fields[0].SetDescription("Some example comment")
	bindings[1].Fields[1].SetName("body")

	if got, want := meth.HttpBindings, bindings; !reflect.DeepEqual(got, want) {
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

	if got, want := svc.GetName(), "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 1; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	meth := svc.Methods[0]
	if got, want := meth.GetName(), "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.RequestType.GetName(), "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meth.ResponseType.GetName(), "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(meth.HttpBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindings := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "get",
					Value: "/ExampleGet",
				},
				&doctree.BindingField{
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	// Have to use SetName/SetDescription methods after declaration since
	// `Name` and `Description are fields of a non-exported embedded struct
	bindings[0].Fields[0].SetName("post")
	bindings[1].Fields[0].SetName("get")
	bindings[1].Fields[0].SetDescription("Some example comment")
	bindings[1].Fields[1].SetName("body")

	if got, want := meth.HttpBindings, bindings; !reflect.DeepEqual(got, want) {
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

	if got, want := svc.GetName(), "Example_Service"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 2; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}
	methone := svc.Methods[0]
	if got, want := methone.GetName(), "Example"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType.GetName(), "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType.GetName(), "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	methtwo := svc.Methods[1]
	if got, want := methtwo.GetName(), "SecondExample"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType.GetName(), "Empty"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType.GetName(), "Empty"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	if got, want := len(methone.HttpBindings), 2; got != want {
		t.Errorf("Http binding count = %#v, want = %#v\n", got, want)
	}

	bindingsone := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "get",
					Value: "/ExampleGet",
				},
				&doctree.BindingField{
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	// Have to use SetName/SetDescription methods after declaration since
	// `Name` and `Description are fields of a non-exported embedded struct
	bindingsone[0].Fields[0].SetName("post")
	bindingsone[1].Fields[0].SetName("get")
	bindingsone[1].Fields[0].SetDescription("Some example comment")
	bindingsone[1].Fields[1].SetName("body")

	if got, want := methone.HttpBindings, bindingsone; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingstwo := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/ExamplePost",
				},
			},
		},
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "get",
					Value: "/SecondExampleGet",
				},
				&doctree.BindingField{
					Kind:  "body",
					Value: "*",
				},
			},
		},
	}
	bindingstwo[0].Fields[0].SetName("post")
	bindingstwo[0].Fields[0].SetDescription("Second binding, this time for post")
	bindingstwo[1].Fields[0].SetName("get")
	bindingstwo[1].Fields[0].SetDescription("Second group of example comments")
	bindingstwo[1].Fields[1].SetName("body")

	if got, want := methtwo.HttpBindings, bindingstwo; !reflect.DeepEqual(got, want) {
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

	if got, want := svc.GetName(), "FlowCombination"; got != want {
		t.Errorf("name = %#v, want = %#v\n", got, want)
	}
	if got, want := len(svc.Methods), 3; got != want {
		t.Errorf("Method count = %#v, want = %#v\n", got, want)
	}

	methone := svc.Methods[0]
	if got, want := methone.GetName(), "RpcEmptyStream"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.RequestType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methone.ResponseType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	methtwo := svc.Methods[1]
	if got, want := methtwo.GetName(), "StreamEmptyRpc"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := methtwo.RequestType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := methtwo.ResponseType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}
	meththree := svc.Methods[2]
	if got, want := meththree.GetName(), "StreamEmptyStream"; got != want {
		t.Errorf("Method name = %#v, want = %#v\n", got, want)
	}
	if got, want := meththree.RequestType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Request type = %#v, want = %#v\n", got, want)
	}
	if got, want := meththree.ResponseType.GetName(), "EmptyProto"; got != want {
		t.Errorf("Response type = %#v, want = %#v\n", got, want)
	}

	bindingsone := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/rpc/empty/stream",
				},
			},
		},
	}
	bindingsone[0].Fields[0].SetName("post")

	if got, want := methone.HttpBindings, bindingsone; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingstwo := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/stream/empty/rpc",
				},
			},
		},
	}
	bindingstwo[0].Fields[0].SetName("post")

	if got, want := methtwo.HttpBindings, bindingstwo; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

	bindingsthree := []*doctree.MethodHttpBinding{
		&doctree.MethodHttpBinding{
			Fields: []*doctree.BindingField{
				&doctree.BindingField{
					Kind:  "post",
					Value: "/stream/empty/stream",
				},
			},
		},
	}
	bindingsthree[0].Fields[0].SetName("post")

	if got, want := meththree.HttpBindings, bindingsthree; !reflect.DeepEqual(got, want) {
		t.Errorf("Http binding contents = %#v, want = %#v\n", got, want)
	}

}
