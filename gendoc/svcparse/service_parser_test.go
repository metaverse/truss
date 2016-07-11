package svcparse

import (
	"strings"
	"testing"
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
	//t.Logf("%v", lex.Buf)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
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
	//t.Logf("%v", lex.Buf)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
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
	t.Logf("%v", lex.Scn.Buf)
	t.Logf("%v", lex.Buf)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
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
	t.Logf("%v", lex.Scn.Buf)
	t.Logf("%v", lex.Buf)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
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
	t.Logf("%v", lex.Scn.Buf)
	t.Logf("%v", lex.Buf)
	svc, err := ParseService(lex)

	if err != nil {
		t.Error(err)
	}
	if svc == nil {
		t.Fatalf("Returned service is nil\n")
	}
}
