package generator

import (
	"bytes"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/handler"
	templateFileAssets "github.com/TuneLab/go-truss/gengokit/template"
	"github.com/TuneLab/go-truss/svcdef"

	"github.com/TuneLab/go-truss/gengokit/gentesthelper"
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

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

func TestApplyTemplateFromPath(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "google.golang.org/genproto/googleapis/api/serviceconfig/annotations.proto";

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
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	conf := gengokit.Config{
		GoPackage: "github.com/TuneLab/go-truss",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := gengokit.NewTemplateExecutor(sd, conf)
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

func svcMethodsNames(methods []*svcdef.ServiceMethod) []string {
	var mNames []string
	for _, m := range methods {
		mNames = append(mNames, m.Name)
	}

	return mNames
}

func stringToTemplateExector(def, importPath string) (*gengokit.TemplateExecutor, error) {
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		return nil, err
	}

	conf := gengokit.Config{
		GoPackage: importPath,
		PBPackage: importPath,
	}

	te, err := gengokit.NewTemplateExecutor(sd, conf)
	if err != nil {
		return nil, err
	}

	return te, nil

}

func TestAllTemplates(t *testing.T) {
	const goPackage = "github.com/TuneLab/go-truss/gengokit"
	const goPBPackage = "github.com/TuneLab/go-truss/gengokit/general-service"

	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "google.golang.org/genproto/googleapis/api/serviceconfig/annotations.proto";

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

		import "google.golang.org/genproto/googleapis/api/serviceconfig/annotations.proto";

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

	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	conf := gengokit.Config{
		GoPackage: "github.com/TuneLab/go-truss/gengokit",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := gengokit.NewTemplateExecutor(sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	sd2, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	te2, err := gengokit.NewTemplateExecutor(sd2, conf)
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
			t.Fatalf("%v failed to format on second identical generation\n\nERROR: %v\nCODE:\n\n%v",
				templFP, err.Error(), string(secondCode))
		}

		if bytes.Compare(firstCode, secondCode) != 0 {
			t.Fatal("Generated code differs after regeneration with same definition\n" + gentesthelper.DiffStrings(string(firstCode), string(secondCode)))
		}

		// store the file to act to pass back to testGenerateResponseFile for third generation
		prevGenMap[templatePathToActual(templFP, te.PackageName)] = bytes.NewReader(secondCode)

		// pass in templateExecutor created from def2
		addRPCCode, err := testGenerateResponseFile(templFP, te2, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on third generation with 1 rpc added\n\nERROR: %v\nCODE:\n\n%v",
				templFP, err.Error(), string(addRPCCode))
		}

		// store the file to act to pass back to testGenerateResponseFile for forth generation
		prevGenMap[templatePathToActual(templFP, te.PackageName)] = bytes.NewReader(addRPCCode)

		// pass in templateExecutor create from def1
		_, err = testGenerateResponseFile(templFP, te, prevGenMap)
		if err != nil {
			t.Fatalf("%v failed to format on forth generation with 1 rpc removed\n\nERROR: %v\nCODE:\n\n%v",
				templFP, err.Error(), string(addRPCCode))
		}
	}
}

func TestUpdateMethods(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "google.golang.org/genproto/googleapis/api/serviceconfig/annotations.proto";

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
		GoPackage: "github.com/TuneLab/go-truss/gengokit",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	te, err := gengokit.NewTemplateExecutor(sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	testHandlerGeneration := func(templPath string) {
		svc.Methods = []*svcdef.ServiceMethod{allMethods[0]}
		firstBytes, err := testGenerateResponseFile(templPath, te, nil)
		if err != nil {
			t.Fatal(err)
		}
		firstCode := strings.TrimSpace(string(firstBytes))

		secondCode, err := renderService(svc, firstCode, te, templPath)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Compare(firstCode, secondCode) != 0 {
			t.Fatal("Generated code differs after regenerated with same definition\n" +
				templPath + "\n" +
				diff(firstCode, secondCode))
		}

		svc.Methods = append(svc.Methods, allMethods[1])

		thirdCode, err := renderService(svc, secondCode, te, templPath)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Compare(secondCode, thirdCode) != -1 {
			t.Fatal("Generated code not longer after regenerated with additional service method\n" +
				templPath + "\n" +
				diff(secondCode, thirdCode))
		}

		// remove the first one rpc
		svc.Methods = svc.Methods[1:]

		forthCode, err := renderService(svc, thirdCode, te, templPath)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Compare(thirdCode, forthCode) != 1 {
			t.Fatal("Generated code not shorter after regenerated with fewer service method\n" +
				templPath + "\n" +
				diff(secondCode, thirdCode))
		}

		svc.Methods = allMethods

		fifthCode, err := renderService(svc, forthCode, te, templPath)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Compare(forthCode, fifthCode) != -1 {
			t.Fatal("Generated code not longer after regenerated with additional service method\n" +
				templPath + "\n" +
				diff(secondCode, thirdCode))
		}
	}
	testHandlerGeneration("NAME-service/handlers/server/server_handler.gotemplate")
}

func diff(a, b string) string {
	return gentesthelper.DiffStrings(
		a,
		b,
	)
}

func renderService(svc *svcdef.Service, prev string, te *gengokit.TemplateExecutor, templPath string) (string, error) {
	h, err := handler.New(svc, strings.NewReader(prev), te.PackageName)
	if err != nil {
		return "", err
	}

	next, err := h.Render(templPath, te)
	if err != nil {
		return "", err
	}

	nextBytes, err := ioutil.ReadAll(next)
	if err != nil {
		return "", err
	}

	nextBytes, err = testFormat(nextBytes)
	if err != nil {
		return "", errors.Wrap(err, "cannot format")
	}

	nextCode := strings.TrimSpace(string(nextBytes))

	return nextCode, nil
}

func testGenerateResponseFile(templFP string, te *gengokit.TemplateExecutor, prevGenMap map[string]io.Reader) ([]byte, error) {
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
