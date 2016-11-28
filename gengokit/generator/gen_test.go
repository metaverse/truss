package generator

import (
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
	log.SetLevel(log.DebugLevel)
	_ = errors.Wrap
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

	te, err := gengokit.NewExecutor(sd, conf)
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

	_, err = testFormat(string(endCode))
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

func stringToTemplateExector(def, importPath string) (*gengokit.Executor, error) {
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		return nil, err
	}

	conf := gengokit.Config{
		GoPackage: importPath,
		PBPackage: importPath,
	}

	te, err := gengokit.NewExecutor(sd, conf)
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

	sd1, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	sd2, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	conf := gengokit.Config{
		GoPackage: "github.com/TuneLab/go-truss/gengokit",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	executor1, err := gengokit.NewExecutor(sd1, conf)
	if err != nil {
		t.Fatal(err)
	}

	executor2, err := gengokit.NewExecutor(sd2, conf)
	if err != nil {
		t.Fatal(err)
	}

	for _, templFP := range templateFileAssets.AssetNames() {
		var prev io.Reader

		firstCode, err := testGenerateResponseFile(executor1, prev, templFP)
		if err != nil {
			t.Fatalf("%s failed to format on first generation\n\nERROR:\n\n%s\n\nCODE:\n\n%s", templFP, err.Error(), firstCode)
		}

		// store the file to pass back to testGenerateResponseFile for second generation
		prev = strings.NewReader(firstCode)

		secondCode, err := testGenerateResponseFile(executor1, prev, templFP)
		if err != nil {
			t.Fatalf("%s failed to format on second identical generation\n\nERROR: %s\nCODE:\n\n%s",
				templFP, err.Error(), secondCode)
		}

		if firstCode != secondCode {
			t.Fatal("Generated code differs after regeneration with same definition\n" + diff(firstCode, secondCode))
		}

		// store the file to pass back to testGenerateResponseFile for third generation
		prev = strings.NewReader(secondCode)

		// pass in Executor created from def2
		addRPCCode, err := testGenerateResponseFile(executor2, prev, templFP)
		if err != nil {
			t.Fatalf("%s failed to format on third generation with 1 rpc added\n\nERROR: %s\nCODE:\n\n%s",
				templFP, err.Error(), addRPCCode)
		}

		// store the file to pass back to testGenerateResponseFile for forth generation
		prev = strings.NewReader(addRPCCode)

		// pass in Executor create from def1
		_, err = testGenerateResponseFile(executor1, prev, templFP)
		if err != nil {
			t.Fatalf("%s failed to format on forth generation with 1 rpc removed\n\nERROR: %s\nCODE:\n\n%s",
				templFP, err.Error(), addRPCCode)
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

	te, err := gengokit.NewExecutor(sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	templPath := handler.ServerFile

	svc.Methods = []*svcdef.ServiceMethod{allMethods[0]}

	firstCode, err := renderService(svc, "", te, templPath)
	if err != nil {
		t.Fatal(err)
	}

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

	if thirdCode <= secondCode {
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

	if forthCode >= thirdCode {
		t.Fatal("Generated code not shorter after regenerated with fewer service method\n" +
			templPath + "\n" +
			diff(thirdCode, forthCode))
	}

	svc.Methods = allMethods

	fifthCode, err := renderService(svc, forthCode, te, templPath)
	if err != nil {
		t.Fatal(err)
	}

	if fifthCode <= forthCode {
		t.Fatal("Generated code not longer after regenerated with additional service method\n" +
			templPath + "\n" +
			diff(forthCode, fifthCode))
	}
}

func renderService(svc *svcdef.Service, prev string, te *gengokit.Executor, templPath string) (string, error) {
	var prevFile io.Reader
	if prev != "" {
		prevFile = strings.NewReader(prev)
	}

	h, err := handler.New(svc, prevFile, te.PackageName)
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

	nextCode, err := testFormat(string(nextBytes))
	if err != nil {
		return "", errors.Wrap(err, "cannot format")
	}

	nextCode = strings.TrimSpace(nextCode)

	return nextCode, nil
}

func diff(a, b string) string {
	return gentesthelper.DiffStrings(
		a,
		b,
	)
}

func testGenerateResponseFile(executor *gengokit.Executor, prev io.Reader, templFP string) (string, error) {
	// apply server_handler.go template
	code, err := generateResponseFile(executor, prev, templFP)
	if err != nil {
		return "", err
	}

	// read the code off the io.Reader
	codeBytes, err := ioutil.ReadAll(code)
	if err != nil {
		return "", err
	}

	// format the code
	formatted, err := testFormat(string(codeBytes))
	if err != nil {
		return string(codeBytes), err
	}

	return formatted, nil
}

// testFormat takes a string representing golang code and attempts to return a
// formated copy of that code.
func testFormat(code string) (string, error) {
	formatted, err := format.Source([]byte(code))

	if err != nil {
		return code, err
	}

	return string(formatted), nil
}
