package httptransport

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/gochipon/truss/gengokit/gentesthelper"
	"github.com/gochipon/truss/svcdef"
)

var (
	_ = spew.Sdump
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestNewMethod(t *testing.T) {
	defStr := `
		syntax = "proto3";

		// General package
		package general;

		import "github.com/gochipon/truss/deftree/googlethirdparty/annotations.proto";

		message SumRequest {
			int64 a = 1;
			int64 b = 2;
			int64 orig_name = 3;
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
	binding := &Binding{
		Label:        "SumZero",
		PathTemplate: "/sum/{a}",
		BasePath:     "/sum/",
		Verb:         "get",
		Fields: []*Field{
			&Field{
				Name:                       "A",
				QueryParamName:             "a",
				CamelName:                  "A",
				LowCamelName:               "a",
				LocalName:                  "ASum",
				Location:                   "path",
				GoType:                     "int64",
				ConvertFunc:                "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				ConvertFuncNeedsErrorCheck: true,
				TypeConversion:             "ASum",
				IsBaseType:                 true,
			},
			&Field{
				Name:                       "B",
				QueryParamName:             "b",
				CamelName:                  "B",
				LowCamelName:               "b",
				LocalName:                  "BSum",
				Location:                   "query",
				GoType:                     "int64",
				ConvertFunc:                "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				ConvertFuncNeedsErrorCheck: true,
				TypeConversion:             "BSum",
				IsBaseType:                 true,
			},
			&Field{
				Name:                       "OrigName",
				QueryParamName:             "orig_name",
				CamelName:                  "OrigName",
				LowCamelName:               "origName",
				LocalName:                  "OrigNameSum",
				Location:                   "query",
				GoType:                     "int64",
				ConvertFunc:                "OrigNameSum, err := strconv.ParseInt(OrigNameSumStr, 10, 64)",
				ConvertFuncNeedsErrorCheck: true,
				TypeConversion:             "OrigNameSum",
				IsBaseType:                 true,
			},
		},
	}
	meth := &Method{
		Name:         "Sum",
		RequestType:  "SumRequest",
		ResponseType: "SumReply",
		Bindings: []*Binding{
			binding,
		},
	}
	binding.Parent = meth

	newMeth := NewMethod(sd.Service.Methods[0])
	if got, want := newMeth, meth; !reflect.DeepEqual(got, want) {
		diff := gentesthelper.DiffStrings(spew.Sdump(got), spew.Sdump(want))
		t.Errorf("got != want; methods differ: %v\n", diff)
	}
}

func TestFuncSourceCode(t *testing.T) {
	_, err := FuncSourceCode(PathParams)
	if err != nil {
		t.Fatalf("Failed to get source code: %s\n", err)
	}
}

func TestAllFuncSourceCode(t *testing.T) {
	_, err := AllFuncSourceCode(PathParams)
	if err != nil {
		t.Fatalf("Failed to get source code: %s\n", err)
	}
}

func TestEnglishNumber(t *testing.T) {
	var cases = []struct {
		i    int
		want string
	}{
		{0, "Zero"},
		{1, "One"},
		{2, "Two"},
		{3, "Three"},
		{4, "Four"},
		{5, "Five"},
		{6, "Six"},
		{7, "Seven"},
		{8, "Eight"},
		{9, "Nine"},

		{11, "OneOne"},
		{22, "TwoTwo"},
		{23, "TwoThree"},
	}

	for _, test := range cases {
		got := EnglishNumber(test.i)
		if got != test.want {
			t.Errorf("Got %v, want %v\n", got, test.want)
		}
	}
}

func TestLowCamelName(t *testing.T) {
	var cases = []struct {
		name, want string
	}{
		{"what", "what"},
		{"example_one", "exampleOne"},
		{"another_example_case", "anotherExampleCase"},
		{"_leading_camel", "xLeadingCamel"},
		{"_a", "xA"},
		{"a", "a"},
	}

	for _, test := range cases {
		got := LowCamelName(test.name)
		if got != test.want {
			t.Errorf("Got %v, want %v\n", got, test.want)
		}
	}
}

func Test_getMuxPathTemplate(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "no pattern",
			path: "/v1/{parent}/books",
			want: "/v1/{parent}/books",
		},
		{
			name: "no *",
			path: "/v1/{parent=shelves}/books",
			want: "/v1/{parent:shelves}/books",
		},
		{
			name: "single *",
			path: "/v1/{parent=shelves/*}/books",
			want: `/v1/{parent:shelves/[^/]+}/books`,
		},
		{
			name: "multiple *",
			path: "/v1/{name=shelves/*/books/*}",
			want: `/v1/{name:shelves/[^/]+/books/[^/]+}`,
		},
		{
			name: "**",
			path: "/v1/shelves/{name=books/**}",
			want: `/v1/shelves/{name:books/.+}`,
		},
		{
			name: "mixed * and **",
			path: "/v1/{name=shelves/*/books/**}",
			want: `/v1/{name:shelves/[^/]+/books/.+}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMuxPathTemplate(tt.path); got != tt.want {
				t.Errorf("getMuxPathTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBinding_PathSections(t *testing.T) {
	tests := []struct {
		name         string
		pathTemplate string
		want         []string
	}{
		{
			name:         "simple",
			pathTemplate: "/sum/{a}",
			want: []string{
				`""`,
				`"sum"`,
				"fmt.Sprint(req.A)",
			},
		},
		{
			name:         "pattern",
			pathTemplate: `/v1/{parent:shelves/[^/]+}/books`,
			want: []string{
				`""`,
				`"v1"`,
				"fmt.Sprint(req.Parent)",
				`"books"`,
			},
		},
		{
			name:         "dot notation",
			pathTemplate: `/v1/{book.name:shelves/[^/]+/books/[^/]+}`,
			want: []string{
				`""`,
				`"v1"`,
				"fmt.Sprint(req.Book.Name)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Binding{
				PathTemplate: tt.pathTemplate,
			}
			if got := b.PathSections(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Binding.PathSections() = %v, want %v", got, tt.want)
			}
		})
	}
}
