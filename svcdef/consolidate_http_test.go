package svcdef

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/tuneinc/truss/svcdef/svcparse"
)

func TestGetPathParams(t *testing.T) {
	binding := &svcparse.HTTPBinding{
		Fields: []*svcparse.Field{
			&svcparse.Field{
				Kind:  "get",
				Value: `"/{a}/{b}"`,
			},
		},
	}
	params := getPathParams(binding)
	if len(params) != 2 {
		t.Fatalf("Params (%v) is length '%v', expected length 2", params, len(params))
	}
	expected := []string{"a", "b"}
	if !reflect.DeepEqual(params, expected) {
		t.Fatalf("Params is %v, expected %v", params, expected)
	}
}

func TestHTTPParams(t *testing.T) {
	goCode := `
package TEST

type EnumType int32

const (
	EnumType_A EnumType = 0
	EnumType_B EnumType = 1
	EnumType_C EnumType = 2
)
type MsgA struct {
	A int64
}
type Thing struct {
	A        *MsgA
	AA       []*MsgA
	C        EnumType
	MapField map[string]*MsgA
}

type MapServer interface {
	GetThing(context.Context, *Thing) (*Thing, error)
}
`
	protoCode := `
syntax = "proto3";
package TEST;
import "github.com/tuneinc/truss/deftree/googlethirdparty/annotations.proto";

enum EnumType {
  A = 0;
  B = 1;
  C = 2;
}

message MsgA {
  int64 A = 1;
}

message Thing {
  MsgA A = 1;
  repeated MsgA AA = 17;
  EnumType C = 18;
  map<string, MsgA> MapField = 19;
}

service Map {
  rpc GetThing (Thing) returns (Thing) {
    option (google.api.http) = {
      get: "/1"
	  body: "*"
    };
  }
}`
	// From code, build our SvcDef
	sd, err := New(map[string]io.Reader{"/tmp/notreal": strings.NewReader(goCode)}, map[string]io.Reader{"/tmp/alsonotreal": strings.NewReader(protoCode)})
	if err != nil {
		t.Fatal(err)
	}

	tmap := newTypeMap(sd)

	rq := sd.Service.Methods[0].RequestType
	bind := sd.Service.Methods[0].Bindings[0]
	if len(bind.Params) != len(tmap["Thing"].Message.Fields) {
		t.Fatalf(
			"Number of http parameters '%v' differs from number of fields on message '%v'",
			len(bind.Params), len(tmap["Thing"].Message.Fields))
	}

	fieldWithName := func(name string) *Field {
		for _, f := range rq.Message.Fields {
			if f.Name == name {
				return f
			}
		}
		return nil
	}

	// Verify that each HTTPParam corresponds to a single field on the RequestType
	for _, param := range bind.Params {
		parmName := param.Field.Name
		fld := fieldWithName(parmName)
		if fld == nil {
			t.Fatalf("HTTPParam with name %q does not have corresponding field on RequestType", parmName)
		}
		if fld != param.Field {
			t.Fatalf("Parameter %q does not refer to the same field as field %q", parmName, fld.Name)
		}
	}

}
