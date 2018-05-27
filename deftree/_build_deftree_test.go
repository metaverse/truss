package deftree

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	// This has to be imported because it modifies the state of `proto` by
	// registering the `google.api.http` extension, allowing us to specify it
	// in the sources below.
	_ "github.com/tuneinc/truss/deftree/googlethirdparty"

	"github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestNewFromString(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "github.com/tuneinc/truss/deftree/googlethirdparty/annotations.proto";

		// RequestMessage is so foo
		message RequestMessage {
		  string input = 1;
		}

		// ResponseMessage is so bar
		message ResponseMessage {
		  string output = 1;
		}

		// ProtoService is a service
		service ProtoService {
		  // ProtoMethod is simple. Like a gopher.
		  rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
			// No {} in path and no body, everything is in the query
			option (google.api.http) = {
			  get: "/route"
			};
		  }
		}
	`

	deftree, err := NewFromString(def, gopath)
	if err != nil {
		t.Error(err)
	}

	if got, want := deftree.GetName(), "general"; got != want {
		t.Errorf("`%v` was deftree package name \n`%v` was wanted", got, want)
	}
}

func TestNewFile(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
		message_type <
			name: "StringMessage"
			field <
				name: "string"
				number: 1
				label: LABEL_OPTIONAL
				type: TYPE_STRING
			>
		>
		service <
			name: "ExampleService"
			method <
				name: "Echo"
				input_type: "StringMessage"
				output_type: "StringMessage"
				options <
					[google.api.http] <
						post: "/v1/example/echo"
						body: "*"
					>
				>
			>
		>
	`
	var fd descriptor.FileDescriptorProto
	if err := proto.UnmarshalText(src, &fd); err != nil {
		t.Fatalf("proto.UnmarshalText(%s, &fd) failed with %v; want success", src, err)
	}

	dt := MicroserviceDefinition{}
	newFile, err := NewFile(&fd, &dt)
	if err != nil {
		t.Fatalf("Error creating new file: %v\n", err)
	}

	msg := ProtoMessage{
		Fields: []*MessageField{
			&MessageField{
				Label:  "LABEL_OPTIONAL",
				Number: 1,
			},
		},
	}
	msg.SetName("StringMessage")
	msg.Fields[0].SetName("string")
	msg.Fields[0].Type.SetName("TYPE_STRING")

	f := &ProtoFile{
		Messages: []*ProtoMessage{
			&msg,
		},
		Services: []*ProtoService{
			&ProtoService{
				FullyQualifiedName: ".example.ExampleService",
				Methods: []*ServiceMethod{
					&ServiceMethod{
						RequestType:  &msg,
						ResponseType: &msg,
					},
				},
			},
		},
	}
	f.SetName("path/to/example.proto")
	f.Services[0].SetName("ExampleService")
	f.Services[0].Methods[0].SetName("Echo")

	if got, want := newFile, f; !reflect.DeepEqual(got, want) {
		t.Errorf("deftree.ProtoFile = \n%#v, want = \n%#v\n", got, want)
	}
}
