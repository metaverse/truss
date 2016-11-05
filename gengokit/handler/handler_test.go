package handler

import (
	"io/ioutil"
	"strings"
	"testing"

	thelper "github.com/TuneLab/go-truss/gengokit/gentesthelper"
	"github.com/TuneLab/go-truss/svcdef"
)

func init() {
	_ = thelper.DiffStrings
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
