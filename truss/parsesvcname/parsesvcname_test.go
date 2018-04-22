package parsesvcname

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// Provide a basic proto file to test that FromPaths returns the name of the
// service in the file at the provided path.
func TestFromPaths(t *testing.T) {
	protoStr := `
	syntax = "proto3";
	package echo;

	import "github.com/tuneinc/truss/deftree/googlethirdparty/annotations.proto";

	service BounceEcho {
	  rpc Echo (EchoRequest) returns (EchoResponse) {
		option (google.api.http) = {
			get: "/echo"
		  };
	  }
	}
	message EchoRequest {
	  string In = 1;
	}
	message EchoResponse {
	  string Out = 1;
	}
	`
	protoDir, err := ioutil.TempDir("", "parsesvcname-test")
	if err != nil {
		t.Fatal("cannot create temp directory to store proto definition: ", err)
	}
	defer os.RemoveAll(protoDir)
	f, err := ioutil.TempFile(protoDir, "trusstest")
	_, err = io.Copy(f, strings.NewReader(protoStr))
	if err != nil {
		t.Fatal("couldn't copy contents of our proto file into the os.File: ", err)
	}
	path := f.Name()
	f.Close()

	svcname, err := FromPaths([]string{os.Getenv("GOPATH")}, []string{path})
	if err != nil {
		t.Fatal("failed to get service name from path: ", err)
	}

	if got, want := svcname, "BounceEcho"; got != want {
		t.Fatalf("got != want; got = %q, want = %q", got, want)
	}
}

// Provide a basic protobuf file to FromReader to ensure it returns the service
// name we expect.
func TestFromReader(t *testing.T) {
	protoStr := `
	syntax = "proto3";
	package echo;

	import "github.com/tuneinc/truss/deftree/googlethirdparty/annotations.proto";

	service BounceEcho {
	  rpc Echo (EchoRequest) returns (EchoResponse) {
		option (google.api.http) = {
			get: "/echo"
		  };
	  }
	}
	message EchoRequest {
	  string In = 1;
	}
	message EchoResponse {
	  string Out = 1;
	}
	`
	svcname, err := FromReaders([]string{os.Getenv("GOPATH")}, []io.Reader{strings.NewReader(protoStr)})
	if err != nil {
		t.Fatal("failed to get service name from path: ", err)
	}

	if got, want := svcname, "BounceEcho"; got != want {
		t.Fatalf("got != want; got = %q, want = %q", got, want)
	}
}

// Ensure that passing a protobuf file that's not importing google annotations
// will function properly.
func TestNoAnnotations(t *testing.T) {
	protoStr := `
	syntax = "proto3";
	package echo;

	service BounceEcho {
	  rpc Echo (EchoRequest) returns (EchoResponse) {}
	}
	message EchoRequest {
	  string In = 1;
	}
	message EchoResponse {
	  string Out = 1;
	}
	`
	svcname, err := FromReaders([]string{os.Getenv("GOPATH")}, []io.Reader{strings.NewReader(protoStr)})
	if err != nil {
		t.Fatal("failed to get service name from path: ", err)
	}

	if got, want := svcname, "BounceEcho"; got != want {
		t.Fatalf("got != want; got = %q, want = %q", got, want)
	}
}

// Test that having a service name which includes an underscore doesn't cause
// problems.
func TestUnderscoreService(t *testing.T) {
	protoStr := `
	syntax = "proto3";
	package echo;

	service foo_bar_test {
	  rpc Echo (EchoRequest) returns (EchoResponse) {}
	}
	message EchoRequest {
	  string In = 1;
	}
	message EchoResponse {
	  string Out = 1;
	}
	`
	svcname, err := FromReaders([]string{os.Getenv("GOPATH")}, []io.Reader{strings.NewReader(protoStr)})
	if err != nil {
		t.Fatal("failed to get service name from path: ", err)
	}

	if got, want := svcname, "FooBarTest"; got != want {
		t.Fatalf("got != want; got = %q, want = %q", got, want)
	}
}

// Test that having a service name which starts with an underscore doesn't
// cause problems.
func TestLeadingUnderscoreService(t *testing.T) {
	protoStr := `
	syntax = "proto3";
	package echo;

	service _Foo_Bar {
	  rpc Echo (EchoRequest) returns (EchoResponse) {}
	}
	message EchoRequest {
	  string In = 1;
	}
	message EchoResponse {
	  string Out = 1;
	}
	`
	svcname, err := FromReaders([]string{os.Getenv("GOPATH")}, []io.Reader{strings.NewReader(protoStr)})
	if err != nil {
		t.Fatal("failed to get service name from path: ", err)
	}

	if got, want := svcname, "XFoo_Bar"; got != want {
		t.Fatalf("got != want; got = %q, want = %q", got, want)
	}
}
