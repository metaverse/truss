package handler

import (
	log "github.com/Sirupsen/logrus"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	//	"github.com/y0ssar1an/q"

	thelper "github.com/TuneLab/go-truss/gengokit/gentesthelper"
	"github.com/TuneLab/go-truss/svcdef"
)

func init() {
	_ = thelper.DiffStrings
	log.SetLevel(log.DebugLevel)

}

func TestServerTempl(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "google/api/annotations.proto";

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
	sd, err := svcdef.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	var he handlerExecutor
	he.Methods = sd.Service.Methods
	he.PackageName = sd.PkgName

	gen, err := applyServerTempl(he)
	if err != nil {
		t.Fatal(err)
	}
	genBytes, err := ioutil.ReadAll(gen)
	const expected = `
		// ProtoMethod implements Service.
		func (s generalService) ProtoMethod(ctx context.Context, in *pb.RequestMessage) (*pb.ResponseMessage, error){
			var resp pb.ResponseMessage
			resp = pb.ResponseMessage{
				// Output:
				}
			return &resp, nil
		}
	`
	a, b, di := thelper.DiffGoCode(string(genBytes), expected)
	if strings.Compare(a, b) != 0 {
		t.Fatalf("Server template different than expected\n %s", di)
	}
}

func TestMRecvTypeString(t *testing.T) {
	values := []string{
		`package p; func NoRecv() {}`, "",
		`package p; func (s Foo) RecvFoo() {}`, "Foo",
		`package p; func (s *Foo) RecvStarFoo() {}`, "Foo",
		`package p; func (s foo.Foo) RecvFooDotFoo() {}`, "foo.Foo",
		`package p; func (s *foo.Foo) RecvStarFooDotFoo() {}`, "foo.Foo",
	}

	for i := 0; i < len(values); i += 2 {
		fnc := parseFuncFromString(values[i], t)
		got := mRecvTypeString(fnc.Recv)
		want := values[i+1]
		if got != want {
			t.Fatalf("Func Recv got: \"%s\", want: \"%s\": for func: %s", got, want, values[i])
		}
	}
}

func TestIsValidFunc(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "google/api/annotations.proto";

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
	sd, err := svcdef.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	m := newMethodMap(sd.Service.Methods)
	const validUnexported = `package p; 
	func init() {}`

	const valid = `package p; 
	func (s generalService) ProtoMethod(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	const invalidRecv = `package p; 
	func (s fooService) ProtoMethod(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	const invalidFuncName = `package p; 
	func (generalService) FOOBAR(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	var in string
	in = validUnexported
	if ok := isValidFunc(parseFuncFromString(in, t), m, sd.PkgName); !ok {
		t.Errorf("Unexported Func considered invalid: %q", in)
	}
	in = valid
	if ok := isValidFunc(parseFuncFromString(in, t), m, sd.PkgName); !ok {
		t.Errorf("Func in service definition with proper recv considered invalid: %q", in)
	}
	in = invalidRecv
	if ok := isValidFunc(parseFuncFromString(in, t), m, sd.PkgName); ok {
		t.Errorf("Func with invalid recv considered valid: %q", in)
	}
	in = invalidFuncName
	if ok := isValidFunc(parseFuncFromString(in, t), m, sd.PkgName); ok {
		t.Errorf("Func with invalid name considered valid: %q", in)
	}
}

func parseFuncFromString(f string, t *testing.T) *ast.FuncDecl {
	fset := token.NewFileSet()
	e, err := parser.ParseFile(fset, "", f, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fnc *ast.FuncDecl
	for _, d := range e.Decls {
		if d, ok := d.(*ast.FuncDecl); ok {
			fnc = d
			break
		}
	}
	if fnc == nil {
		t.Fatalf("No function found in: %q", f)
	}
	return fnc
}
