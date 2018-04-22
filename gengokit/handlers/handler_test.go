package handlers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/tuneinc/truss/gengokit"
	helper "github.com/tuneinc/truss/gengokit/gentesthelper"
	"github.com/tuneinc/truss/svcdef"
)

var gopath []string
var diff = helper.DiffStrings
var testFormat = helper.TestFormat

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestServerMethsTempl(t *testing.T) {
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

		// Proto is a service
		service Proto {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
		}
	`
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	var he handlerData
	he.Methods = sd.Service.Methods
	he.ServiceName = sd.Service.Name

	gen, err := applyServerMethsTempl(he)
	if err != nil {
		t.Fatal(err)
	}
	genBytes, err := ioutil.ReadAll(gen)
	const expected = `
		// ProtoMethod implements Service.
		func (s protoService) ProtoMethod(ctx context.Context, in *pb.RequestMessage) (*pb.ResponseMessage, error){
			var resp pb.ResponseMessage
			resp = pb.ResponseMessage{
				// Output:
				}
			return &resp, nil
		}
	`
	a, b, di := helper.DiffGoCode(string(genBytes), expected)
	if strings.Compare(a, b) != 0 {
		t.Fatalf("Server method template output different than expected\n %s", di)
	}
}

func TestApplyServerTempl(t *testing.T) {
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

		// Proto is a service
		service Proto {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
		}
	`
	conf := gengokit.Config{
		GoPackage: "github.com/tuneinc/truss/gengokit/general-service",
		PBPackage: "github.com/tuneinc/truss/gengokit/general-service",
	}
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}
	te, err := gengokit.NewData(sd, conf)

	gen, err := applyServerTempl(te)
	genBytes, err := ioutil.ReadAll(gen)
	expected := `
		package handlers

		import (
			"context"

			pb "github.com/tuneinc/truss/gengokit/general-service"
		)

		// NewService returns a naïve, stateless implementation of Service.
		func NewService() pb.ProtoServer {
			return protoService{}
		}

		type protoService struct{}

		// ProtoMethod implements Service.
		func (s protoService) ProtoMethod(ctx context.Context, in *pb.RequestMessage) (*pb.ResponseMessage, error) {
			var resp pb.ResponseMessage
			resp = pb.ResponseMessage{
			// Output:
			}
			return &resp, nil
		}
	`
	a, b, di := helper.DiffGoCode(string(genBytes), expected)
	if strings.Compare(a, b) != 0 {
		t.Fatalf("Server template output different than expected\n %s", di)
	}
}

func TestRecvTypeToString(t *testing.T) {
	values := []string{
		`package p; func NoRecv() {}`, "",
		`package p; func (s Foo) RecvFoo() {}`, "Foo",
		`package p; func (s *Foo) RecvStarFoo() {}`, "*Foo",
		`package p; func (s foo.Foo) RecvFooDotFoo() {}`, "foo.Foo",
		`package p; func (s *foo.Foo) RecvStarFooDotFoo() {}`, "*foo.Foo",
	}

	for i := 0; i < len(values); i += 2 {
		fnc := parseFuncFromString(values[i], t)
		got := recvTypeToString(fnc.Recv)
		want := values[i+1]
		if got != want {
			t.Errorf("Func Recv got: \"%s\", want: \"%s\": for func: %s", got, want, values[i])
		}
	}
}

func TestIsValidFunc(t *testing.T) {
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

		// Proto is a service
		service Proto {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
		}
	`
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	m := newMethodMap(sd.Service.Methods)
	const validUnexported = `package p;
	func init() {}`

	const valid = `package p;
	func (s protoService) ProtoMethod(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	const invalidRecv = `package p;
	func (s fooService) ProtoMethod(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	const invalidFuncName = `package p;
	func (generalService) FOOBAR(context.Context, pb.RequestMessage) (pb.ResponseMessage, error) {}`

	svcName := strings.ToLower(sd.Service.Name)

	var in string
	in = validUnexported
	if ok := isValidFunc(parseFuncFromString(in, t), m, svcName); !ok {
		t.Errorf("Unexported Func considered invalid: %q", in)
	}
	in = valid
	if ok := isValidFunc(parseFuncFromString(in, t), m, svcName); !ok {
		t.Errorf("Func in service definition with proper recv considered invalid: %q", in)
	}
	in = invalidRecv
	if ok := isValidFunc(parseFuncFromString(in, t), m, svcName); ok {
		t.Errorf("Func with invalid recv considered valid: %q", in)
	}
	in = invalidFuncName
	if ok := isValidFunc(parseFuncFromString(in, t), m, svcName); ok {
		t.Errorf("Func with invalid name considered valid: %q", in)
	}
}

func TestPruneDecls(t *testing.T) {
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

		// Proto is a service
		service Proto {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
			// ProtoMethodAgain is simple. Like a gopher again.
			rpc ProtoMethodAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route2"
				};
			}
			// ProtoMethodAgainAgain is simple. Like a gopher again again.
			rpc ProtoMethodAgainAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route3"
				};
			}
		}
	`
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	m := newMethodMap(sd.Service.Methods)

	prev := `
		package handlers

		import (
			"context"

			pb "github.com/tuneinc/truss/gengokit/general-service"
		)

		// NewService returns a naïve, stateless implementation of Service.
		func NewService() pb.ProtoServer {
			return protoService{}
		}

		type protoService struct{}

		func init() {
			//FOOING
		}

		// ProtoMethod implements Service.
		func (s protoService) ProtoMethod(ctx context.Context, in *pb.RequestMessage) (*pb.ResponseMessage, error) {
			var resp pb.ResponseMessage
			resp = pb.ResponseMessage{
			// Output:
			}
			return &resp, nil
		}

		// FOOBAR implements Service.
		func (s protoService) FOOBAR(ctx context.Context, in *pb.RequestMessage) (*pb.ResponseMessage, error) {
			var resp pb.ResponseMessage
			resp = pb.ResponseMessage{
			// Output:
			}
			return &resp, nil
		}
	`
	f := parseASTFromString(prev, t)
	lenDeclsBefore := len(f.Decls)
	lenMMapBefore := len(m)

	newDecls := m.pruneDecls(f.Decls, strings.ToLower(sd.Service.Name))

	lenDeclsAfter := len(newDecls)
	lenMMapAfter := len(m)

	if lenDeclsBefore-1 != lenDeclsAfter {
		t.Fatalf("Prune did update Decls as expected; got: %d, want: %d", lenDeclsBefore-1, lenDeclsAfter)
	}

	if lenMMapBefore-1 != lenMMapAfter {
		t.Fatalf("Prune did update mMap as expected; got: %d, want: %d", lenMMapBefore-1, lenMMapAfter)
	}
}

func TestUpdatePBFieldType(t *testing.T) {
	values := []string{
		`*pb.Old`, "New", "*pb.New",
		`pb.Old`, "New", "pb.New",
		`Old`, "New", "Old",
	}
	for i := 0; i < len(values); i += 3 {
		exp, err := parser.ParseExpr(values[i])
		if err != nil {
			t.Error(err)
		}
		updatePBFieldType(exp, values[i+1])
		got := exprString(exp)
		want := values[i+2]
		if got != want {
			t.Errorf("Func Recv got: \"%s\", want: \"%s\": for func: %s", got, want, values[i])
		}
	}
}

func TestUpdateMethods(t *testing.T) {
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

		// RequestMessage is so foo
		message DifferentRequest {
			float green = 1;
		}

		message DifferentResponse {
			int64 blue = 1;
		}

		// Proto is a service
		service Proto {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
			// ProtoMethodAgain is simple. Like a gopher again.
			rpc ProtoMethodAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route2"
				};
			}
			// ProtoMethodAgainAgain is simple. Like a gopher again again.
			rpc ProtoMethodAgainAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route2"
				};
			}
		}
	`

	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	svc := sd.Service
	allMethods := svc.Methods

	conf := gengokit.Config{
		GoPackage: "github.com/tuneinc/truss/gengokit",
		PBPackage: "github.com/tuneinc/truss/gengokit/general-service",
	}

	te, err := gengokit.NewData(sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	svc.Methods = []*svcdef.ServiceMethod{allMethods[0]}

	firstCode, err := renderService(svc, "", te)
	if err != nil {
		t.Fatal(err)
	}

	secondCode, err := renderService(svc, firstCode, te)
	if err != nil {
		t.Fatal(err)
	}

	if len(firstCode) != len(secondCode) {
		t.Fatal("Generated service differs after regenerated with same definition\n" +
			diff(firstCode, secondCode))
	}

	svc.Methods = append(svc.Methods, allMethods[1])

	thirdCode, err := renderService(svc, secondCode, te)
	if err != nil {
		t.Fatal(err)
	}

	if len(thirdCode) <= len(secondCode) {
		t.Fatal("Generated service not longer after regenerated with additional service method\n" +
			diff(secondCode, thirdCode))
	}

	// remove the first one rpc
	svc.Methods = svc.Methods[1:]

	forthCode, err := renderService(svc, thirdCode, te)
	if err != nil {
		t.Fatal(err)
	}

	if len(forthCode) >= len(thirdCode) {
		t.Fatal("Generated service not shorter after regenerated with fewer service method\n" +
			diff(thirdCode, forthCode))
	}

	svc.Methods = allMethods

	fifthCode, err := renderService(svc, forthCode, te)
	if err != nil {
		t.Fatal(err)
	}

	if len(fifthCode) <= len(forthCode) {
		t.Fatal("Generated service not longer after regenerated with additional service method\n" +
			diff(forthCode, fifthCode))
	}
}

// renderService takes in a previous file as a string and returns the generated
// service file as a string. This helper method exists because the logic for
// reading the io.Reader to a string is repeated.
func renderService(svc *svcdef.Service, prev string, data *gengokit.Data) (string, error) {
	var prevFile io.Reader
	if prev != "" {
		prevFile = strings.NewReader(prev)
	}

	h, err := New(svc, prevFile)
	if err != nil {
		return "", err
	}

	next, err := h.Render(ServerHandlerPath, data)
	if err != nil {
		return "", err
	}

	nextBytes, err := ioutil.ReadAll(next)
	if err != nil {
		return "", err
	}

	nextCode, err := testFormat(string(nextBytes))
	if err != nil {
		return "", errors.Wrap(err, "cannot format")
	}

	nextCode = strings.TrimSpace(nextCode)

	return nextCode, nil
}

func parseASTFromString(s string, t *testing.T) *ast.File {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", s, 0)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func parseFuncFromString(f string, t *testing.T) *ast.FuncDecl {
	file := parseASTFromString(f, t)
	var fnc *ast.FuncDecl
	for _, d := range file.Decls {
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
