package middlewares

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TuneLab/go-truss/gengokit"
	thelper "github.com/TuneLab/go-truss/gengokit/gentesthelper"
	"github.com/TuneLab/go-truss/svcdef"
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestNewServiceMiddleware(t *testing.T) {
	var wantService = `
		package middlewares

		import (
		pb "github.com/TuneLab/go-truss/gengokit/general-service"
		)

		func WrapService(in pb.ProtoServiceServer) pb.ProtoServiceServer {
			return in
		}
	`

	_, data, err := generalService()
	if err != nil {
		t.Fatal(err)
	}

	middleware := New()
	service, err := middleware.Render(ServicePath, data)
	if err != nil {
		t.Fatal(err)
	}

	serviceBytes, err := ioutil.ReadAll(service)
	if err != nil {
		t.Fatal(err)
	}

	wantFormatted, serviceFormatted, diff := thelper.DiffGoCode(wantService, string(serviceBytes))
	if wantFormatted != serviceFormatted {
		t.Fatalf("Serivce middleware different than expected:\n\n%s", diff)
	}
}

func TestNewEndpointMiddleware(t *testing.T) {
	var wantEndpoints = `
		package middlewares

		import (
			_ "github.com/go-kit/kit/endpoint"

			svc "github.com/TuneLab/go-truss/gengokit/general-service/generated"
		)

		// WrapEndpoints accepts the service's entire collection of endpoints, so that a
		// set of middlewares can be wrapped around every middleware (e.g., access
		// logging and instrumentation), and others wrapped selectively around some
		// endpoints and not others (e.g., endpoints requiring authenticated access).
		// Note that the final middleware applied will be the outermost middleware
		// (i.e. applied first)
		func WrapEndpoints(in svc.Endpoints) svc.Endpoints {

			// Pass in the middlewares you want applied to every endpoint.
			// optionally pass in endpoints by name that you want to be excluded
			// e.g.
			// in.WrapAll(authMiddleware, "Status", "Ping")

			// How to apply a middleware to a single endpoint.
			// in.ExampleEndpoint = authMiddleware(in.ExampleEndpoint)

			return in
		}
	`

	_, data, err := generalService()
	if err != nil {
		t.Fatal(err)
	}

	middleware := New()
	endpoints, err := middleware.Render(EndpointsPath, data)
	if err != nil {
		t.Fatal(err)
	}

	endpointsBytes, err := ioutil.ReadAll(endpoints)
	if err != nil {
		t.Fatal(err)
	}

	wantFormatted, endpointFormatted, diff := thelper.DiffGoCode(wantEndpoints, string(endpointsBytes))
	if wantFormatted != endpointFormatted {
		t.Fatalf("Endpoints middleware different than expected:\n\n%s", diff)
	}
}

func TestRenderPrevService(t *testing.T) {
	var wantService = `
		package middlewares

		import (
			pb "github.com/TuneLab/go-truss/gengokit/general-service"
		)

		func WrapService(in pb.ProtoServiceServer) pb.ProtoServiceServer {
			return in
		}

		func FooBar() error {
			return nil
		}
	`
	_, data, err := generalService()
	if err != nil {
		t.Fatal(err)
	}

	middleware := New()

	middleware.LoadService(strings.NewReader(wantService))

	service, err := middleware.Render(ServicePath, data)
	if err != nil {
		t.Fatal(err)
	}

	serviceBytes, err := ioutil.ReadAll(service)
	if err != nil {
		t.Fatal(err)
	}

	wantFormatted, serviceFormatted, diff := thelper.DiffGoCode(wantService, string(serviceBytes))
	if wantFormatted != serviceFormatted {
		t.Fatalf("Sevice middleware modified unexpectedly:\n\n%s", diff)
	}
}

func TestRenderPrevEndpoints(t *testing.T) {
	var wantEndpoints = `
		package middlewares

		import (
			"github.com/go-kit/kit/endpoint"
			svc "github.com/TuneLab/go-truss/gengokit/general-service/generated"
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

	middleware := New()

	middleware.LoadEndpoints(strings.NewReader(wantEndpoints))

	endpoints, err := middleware.Render(EndpointsPath, data)
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

	middleware := New()

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

		import "github.com/TuneLab/go-truss/deftree/googlethirdparty/annotations.proto";

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
		GoPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
		PBPackage: "github.com/TuneLab/go-truss/gengokit/general-service",
	}

	data, err := gengokit.NewData(sd, conf)
	if err != nil {
		return nil, nil, err
	}

	return sd, data, nil
}
