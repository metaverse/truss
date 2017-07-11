package deftree

import (
	"testing"
)

// Test to ensure that placing comments within an Enum functions correctly.
func TestCommentedEnum(t *testing.T) {
	// Create our request, then assemble a basic deftree
	defStr := `
		syntax = "proto3";
		package general;
		import "github.com/TuneLab/truss/deftree/googlethirdparty/annotations.proto";

		enum FooBarBaz {
			// This is my comment, this is my note
		   FOO = /* is this even valid */ 0;
		   BAR = 1; // are we allowed
		   BAZ = 2;
		}
		message SumRequest {
			FooBarBaz a = 1;
			int64 b = 2;
		}

		service SumSvc {
			rpc Sum(SumRequest) returns (SumRequest) {
				option (google.api.http) = {
					get: "/sum/{a}"
				};
			}
		}
	`
	dt, err := NewFromString(defStr, gopath)
	md := dt.(*MicroserviceDefinition)
	if err != nil {
		t.Fatal(err)
	}

	if md == nil {
		t.Fatalf("The returned Deftree is nil")
	}
	got := md.Files[0].Enums[0].Values[0].Description
	want := "This is my comment, this is my note"
	if got != want {
		t.Fatalf("Comment found in Enum is not expected; got = %q, want = %q", got, want)
	}
}
