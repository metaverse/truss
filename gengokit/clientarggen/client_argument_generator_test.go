package clientarggen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/tuneinc/truss/gengokit/gentesthelper"
	"github.com/tuneinc/truss/svcdef"
)

var (
	spw = spew.ConfigState{
		Indent: "   ",
	}
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestNewClientServiceArgs(t *testing.T) {
	defStr := `
		syntax = "proto3";

		// General package
		package general;

		import "github.com/tuneinc/truss/deftree/googlethirdparty/annotations.proto";

		message SumRequest {
			repeated int64 a = 1;
			int64 b = 2;
		}

		message SumReply {
			int64 v = 1;
			string err = 2;
		}

		service SumSvc {
			rpc Sum(SumRequest) returns (SumReply) {
				option (google.api.http) = {
					get: "/sum/{a}"
				};
			}
		}
	`
	sd, err := svcdef.NewFromString(defStr, gopath)
	if err != nil {
		t.Fatal(err, "Failed to create a service from the definition string")
	}
	csa := New(sd.Service)

	expected := &ClientServiceArgs{
		MethArgs: map[string]*MethodArgs{
			"Sum": &MethodArgs{
				Args: []*ClientArg{
					&ClientArg{
						Name:            "a",
						FlagName:        "a",
						FlagArg:         "flagASum",
						FlagType:        "string",
						FlagConvertFunc: "flagASum = fsSum.String(\"a\", \"\", \"\")",
						GoArg:           "ASum",
						GoType:          "[]int64",
						GoConvertInvoc:  "\nvar ASum []int64\nif flagASum != nil && len(*flagASum) > 0 {\n\terr = json.Unmarshal([]byte(*flagASum), &ASum)\n\tif err != nil {\n\t\tpanic(errors.Wrapf(err, \"unmarshalling ASum from %v:\", flagASum))\n\t}\n}\n",
						IsBaseType:      true,
						Repeated:        true,
					},
					&ClientArg{

						Name:            "b",
						FlagName:        "b",
						FlagArg:         "flagBSum",
						FlagType:        "int64",
						FlagConvertFunc: "flagBSum = fsSum.Int64(\"b\", 0, \"\")",
						GoArg:           "BSum",
						GoType:          "int64",
						GoConvertInvoc:  "BSum := *flagBSum",
						IsBaseType:      true,
						Repeated:        false,
					},
				},
			},
		},
	}
	if got, want := csa, expected; !reflect.DeepEqual(got, want) {
		t.Errorf(gentesthelper.DiffStrings(spw.Sdump(got), spw.Sdump(want)))
	}
}
