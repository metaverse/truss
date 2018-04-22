package handlers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuneinc/truss/gengokit"
	thelper "github.com/tuneinc/truss/gengokit/gentesthelper"
	"github.com/tuneinc/truss/svcdef"
)

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestRenderPrevEndpoints(t *testing.T) {
	var wantEndpoints = `
		package middlewares

		import (
			"github.com/go-kit/kit/endpoint"
			"github.com/tuneinc/truss/gengokit/general-service/svc"
		)

		// WrapEndpoint will be called individually for all endpoints defined in
		// the service. Implement this with the middlewares you want applied to
		// every endpoint.
		func WrapEndpoint(in endpoint.Endpoint) endpoint.Endpoint {
			return in
		}

		// WrapEndpoints takes the service's entire collection of endpoints. This
		// function can be used to apply middlewares selectively to some endpoints,
		// but not others, like protecting some endpoints with authentication.
		func WrapEndpoints(in svc.Endpoints) svc.Endpoints {
			return in
		}

		func BarFoo(err error) bool {
			if err != nil {
				return true
			}
			return false
		}
	`

	_, data, err := generalService()
	if err != nil {
		t.Fatal(err)
	}

	middleware := NewMiddlewares()

	middleware.Load(strings.NewReader(wantEndpoints))

	endpoints, err := middleware.Render(MiddlewaresPath, data)
	if err != nil {
		t.Fatal(err)
	}

	endpointsBytes, err := ioutil.ReadAll(endpoints)
	if err != nil {
		t.Fatal(err)
	}

	wantFormatted, endpointFormatted, diff := thelper.DiffGoCode(wantEndpoints, string(endpointsBytes))
	if wantFormatted != endpointFormatted {
		t.Fatalf("Endpoints middleware modified unexpectedly:\n\n%s", diff)
	}
}

func TestRenderUnknownFile(t *testing.T) {
	_, data, err := generalService()
	if err != nil {
		t.Fatal(err)
	}

	middleware := NewMiddlewares()

	_, err = middleware.Render("not/valid/file.go", data)
	if err == nil {
		t.Fatalf("This should have produced an error, but didn't")
	}
}

func generalService() (*svcdef.Svcdef, *gengokit.Data, error) {
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
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		return nil, nil, err
	}
	conf := gengokit.Config{
		GoPackage: "github.com/tuneinc/truss/gengokit/general-service",
		PBPackage: "github.com/tuneinc/truss/gengokit/general-service",
	}

	data, err := gengokit.NewData(sd, conf)
	if err != nil {
		return nil, nil, err
	}

	return sd, data, nil
}
