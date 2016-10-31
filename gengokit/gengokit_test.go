package gengokit

import (
	"bytes"
	"go/format"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/svcdef"

	"github.com/TuneLab/go-truss/gengokit/config"
	templateFileAssets "github.com/TuneLab/go-truss/gengokit/template"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.FatalLevel)
}

func TestTemplatePathToActual(t *testing.T) {

	pathToWants := map[string]string{
		"NAME-service/":                "package-service/",
		"NAME-service/test.gotemplate": "package-service/test.go",
		"NAME-service/NAME-server":     "package-service/package-server",
	}

	for path, want := range pathToWants {
		if got := templatePathToActual(path, "package"); got != want {
			t.Fatalf("\n`%v` got\n`%v` wanted", got, want)
		}
	}
}

func TestNewTemplateExecutor(t *testing.T) {
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
	dt, err := deftree.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	conf := config.Config{
		GoPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := newTemplateExecutor(dt, sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := te.PackageName, dt.GetName(); got != want {
		t.Fatalf("\n`%v` was PackageName\n`%v` was wanted", got, want)
	}
}

func TestGetProtoService(t *testing.T) {
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
	dt, err := deftree.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	svc, err := getProtoService(dt)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := svc.Name, "ProtoService"; got != want {
		t.Fatalf("\n`%v` was service name\n`%v` was wanted", got, want)
	}

	if got, want := svc.Methods[0].Name, "ProtoMethod"; got != want {
		t.Fatalf("\n`%v` was rpc in service\n`%v` was wanted", got, want)
	}
}

func TestApplyTemplateFromPath(t *testing.T) {
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
	dt, err := deftree.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	conf := config.Config{
		GoPackage: "github.com/TuneLab/go-truss",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := newTemplateExecutor(dt, sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	end, err := applyTemplateFromPath("NAME-service/generated/endpoints.gotemplate", te)
	if err != nil {
		t.Fatal(err)
	}

	endCode, err := ioutil.ReadAll(end)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testFormat(endCode)
	if err != nil {
		t.Fatal(err)
	}

}

func TestTrimTemplateExecutorServiceFuncs(t *testing.T) {
	const goPackage = "github.com/TuneLab/go-truss/gengokit"

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
			// ProtoMethodAgain is simple. Like a gopher again.
			rpc ProtoMethodAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route2"
				};
			}
			// ProtoMethodAgain is simple. Like a gopher again.
			rpc ProtoMethodAgainAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route3"
				};
			}
		}
	`

	defMethNames := map[string]bool{
		"ProtoMethod":           true,
		"ProtoMethodAgain":      true,
		"ProtoMethodAgainAgain": true,
	}

	te, err := stringToTemplateExector(def, goPackage)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(te.Service.Methods), 3; got != want {
		t.Fatalf("Got %d methods, wanted %d", got, want)
	}

	teEmpty := te.trimServiceFuncs(defMethNames)
	if got, want := len(teEmpty.Service.Methods), 0; got != want {
		t.Fatalf("Got %d methods, wanted %d", got, want)
	}

	defMethNames["ProtoMethodAgain"] = false
	teOnlyAgain := te.trimServiceFuncs(defMethNames)
	if got, want := len(teOnlyAgain.Service.Methods), 1; got != want {
		t.Fatalf("Got %d methods, wanted %d", got, want)
	}

	teOnlyAgainMeths := svcMethodsNames(teOnlyAgain.Service.Methods)
	got := teOnlyAgainMeths[0]
	want := "ProtoMethodAgain"
	if got != want {
		t.Fatalf("Got `%s` method, wanted `%s`", got, want)
	}

	emptyMeths := make(map[string]bool, 0)
	teFull := te.trimServiceFuncs(emptyMeths)
	if got, want := len(teFull.Service.Methods), 3; got != want {
		t.Fatalf("Got %d methods, wanted %d", got, want)
	}

}

func svcMethodsNames(methods []*svcdef.ServiceMethod) []string {
	var mNames []string
	for _, m := range methods {
		mNames = append(mNames, m.Name)
	}

	return mNames
}

func stringToTemplateExector(def, importPath string) (*templateExecutor, error) {
	dt, err := deftree.NewFromString(def)
	if err != nil {
		return nil, err
	}
	sd, err := svcdef.NewFromString(def)
	if err != nil {
		return nil, err
	}

	conf := config.Config{
		GoPackage: importPath,
		PBPackage: importPath,
	}

	te, err := newTemplateExecutor(dt, sd, conf)
	if err != nil {
		return nil, err
	}

	return te, nil

}

func TestUpdateServerMethods(t *testing.T) {
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
	dt, err := deftree.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	conf := config.Config{
		GoPackage: "github.com/TuneLab/go-truss/gengokit",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := newTemplateExecutor(dt, sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	// apply server_handler.go template
	sh, err := applyTemplateFromPath("NAME-service/handlers/server/server_handler.gotemplate", te)
	if err != nil {
		t.Fatal(err)
	}

	// read the code off the io.Reader
	shCode, err := ioutil.ReadAll(sh)
	if err != nil {
		t.Fatal(err)
	}

	// format the code
	shCode, err = testFormat(shCode)
	if err != nil {
		t.Fatal(err)
	}

	// updateServerMethods with the same templateExecutor
	same, err := updateServerMethods(bytes.NewReader(shCode), te)
	if err != nil {
		t.Fatal(err)
	}

	// read the new code off the io.Reader
	sameCode, err := ioutil.ReadAll(same)
	if err != nil {
		t.Fatal(err)
	}

	// format that new code
	sameCode, err = testFormat(sameCode)
	if err != nil {
		t.Fatal(err)
	}

	// make sure the code is the same
	if bytes.Compare(shCode, sameCode) != 0 {
		t.Fatalf("\n__BEFORE__\n\n%s\n\n__AFTER__\n\n%s\n\nCode before and after updating differs",
			string(shCode), string(sameCode))
	}
}

func TestAllTemplates(t *testing.T) {
	const goPackage = "github.com/TuneLab/go-truss/gengokit"
	const goPBPackage = "github.com/TuneLab/go-truss/gengokit/general-service"

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

	const def2 = `
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
			// ProtoMethodAgain is simple. Like a gopher again.
			rpc ProtoMethodAgain (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route2"
				};
			}
		}
	`

	sd, err := svcdef.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}
	dt, err := deftree.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	conf := config.Config{
		GoPackage: "github.com/TuneLab/go-truss/gengokit",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := newTemplateExecutor(dt, sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	dt2, err := deftree.NewFromString(def2)
	if err != nil {
		t.Fatal(err)
	}
	sd2, err := svcdef.NewFromString(def)
	if err != nil {
		t.Fatal(err)
	}

	te2, err := newTemplateExecutor(dt2, sd2, conf)
	if err != nil {
		t.Fatal(err)
	}

	for _, templFP := range templateFileAssets.AssetNames() {
		// skip the partial templates
		if filepath.Ext(templFP) != ".gotemplate" {
			continue
		}
		prevGenMap := make(map[string]io.Reader)

		firstCode, err := testGenerateResponseFile(templFP, te, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on first generation\n\nERROR:\n\n%v\n\nCODE:\n\n%v", templFP, err.Error(), string(firstCode))
		}

		// store the file to act to pass back to testGenerateResponseFile for second generation
		prevGenMap[templatePathToActual(templFP, te.PackageName)] = bytes.NewReader(firstCode)

		secondCode, err := testGenerateResponseFile(templFP, te, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on second identical generation\n\nERROR:\n\n%v\n\nCODE:\n\n%v", templFP, err.Error(), string(secondCode))
		}

		if bytes.Compare(firstCode, secondCode) != 0 {
			t.Fatalf("\n__BEFORE__\n\n%v\n\n__AFTER\n\n%v\n\nCode differs after being regenerated with same definition file",
				string(firstCode), string(secondCode))
		}

		// store the file to act to pass back to testGenerateResponseFile for third generation
		prevGenMap[templatePathToActual(templFP, te.PackageName)] = bytes.NewReader(secondCode)

		// pass in templateExecutor created from def2
		addRPCCode, err := testGenerateResponseFile(templFP, te2, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on third generation with 1 rpc added\n\nERROR:\n\n%v\n\nCODE:\n\n%v", templFP, err.Error(), string(addRPCCode))
		}

		// store the file to act to pass back to testGenerateResponseFile for forth generation
		prevGenMap[templatePathToActual(templFP, te.PackageName)] = bytes.NewReader(addRPCCode)

		// pass in templateExecutor create from def1
		_, err = testGenerateResponseFile(templFP, te, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on forth generation with 1 rpc removed\n\nERROR:\n\n%v\n\nCODE:\n\n%v", templFP, err.Error(), string(addRPCCode))
		}
	}
}

func testGenerateResponseFile(templFP string, te *templateExecutor, prevGenMap map[string]io.Reader) ([]byte, error) {
	// apply server_handler.go template
	code, err := generateResponseFile(templFP, te, prevGenMap)
	if err != nil {
		return nil, err
	}

	// read the code off the io.Reader
	codeBytes, err := ioutil.ReadAll(code)
	if err != nil {
		return nil, err
	}

	// format the code
	formatted, err := testFormat(codeBytes)
	if err != nil {
		return codeBytes, err
	}

	return formatted, nil
}

// testFormat takes a string representing golang code and attempts to return a
// formated copy of that code.
func testFormat(code []byte) ([]byte, error) {
	formatted, err := format.Source(code)

	if err != nil {
		return code, err
	}

	return formatted, nil
}
