package httptransport

import (
	"reflect"
	"testing"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/davecgh/go-spew/spew"
)

var (
	_ = spew.Sdump
)

func TestNewMethod(t *testing.T) {
	dmeth := deftree.ServiceMethod{
		Name: "Sum",
		RequestType: &deftree.ProtoMessage{
			Name: "SumRequest",
			//Description: "The sum request contains two parameters.",
			Fields: []*deftree.MessageField{
				&deftree.MessageField{
					Name:   "a",
					Number: 1,
					Label:  "LABEL_OPTIONAL",
					Type: deftree.FieldType{
						Name: "TYPE_INT64",
					},
				},
				&deftree.MessageField{
					Name:   "b",
					Number: 2,
					Label:  "LABEL_OPTIONAL",
					Type: deftree.FieldType{
						Name: "TYPE_INT64",
					},
				},
			},
		},
		ResponseType: &deftree.ProtoMessage{
			Name: "SumReply",
			Fields: []*deftree.MessageField{
				&deftree.MessageField{
					Name:   "v",
					Number: 1,
					Label:  "LABEL_OPTIONAL",
					Type: deftree.FieldType{
						Name: "TYPE_INT64",
					},
				},
				&deftree.MessageField{
					Name:   "err",
					Number: 2,
					Label:  "LABEL_OPTIONAL",
					Type: deftree.FieldType{
						Name: "TYPE_STRING",
					},
				},
			},
		},
		HttpBindings: []*deftree.MethodHttpBinding{
			&deftree.MethodHttpBinding{
				Verb: "get",
				Path: "/sum/{a}",
				Fields: []*deftree.BindingField{
					&deftree.BindingField{
						Name:  "get",
						Kind:  "get",
						Value: "/sum/{a}",
					},
				},
				Params: []*deftree.HttpParameter{
					&deftree.HttpParameter{
						Name:     "a",
						Location: "path",
						Type:     "TYPE_INT64",
					},
					&deftree.HttpParameter{
						Name:     "b",
						Location: "query",
						Type:     "TYPE_INT64",
					},
				},
			},
		},
	}
	binding := &Binding{
		Label:        "SumZero",
		PathTemplate: "/sum/{a}",
		BasePath:     "/sum/",
		Verb:         "get",
		Fields: []*Field{
			&Field{
				Name:          "a",
				CamelName:     "A",
				LowCamelName:  "a",
				LocalName:     "ASum",
				Location:      "path",
				ProtobufType:  "TYPE_INT64",
				GoType:        "int64",
				ProtobufLabel: "LABEL_OPTIONAL",
				ConvertFunc:   "ASum, err := strconv.ParseInt(ASumStr, 10, 64)",
				IsBaseType:    true,
			},
			&Field{
				Name:          "b",
				CamelName:     "B",
				LowCamelName:  "b",
				LocalName:     "BSum",
				Location:      "query",
				ProtobufType:  "TYPE_INT64",
				GoType:        "int64",
				ProtobufLabel: "LABEL_OPTIONAL",
				ConvertFunc:   "BSum, err := strconv.ParseInt(BSumStr, 10, 64)",
				IsBaseType:    true,
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

	newMeth := NewMethod(&dmeth)
	if got, want := newMeth, meth; !reflect.DeepEqual(got, want) {
		t.Errorf("methods differ;\ngot  = %+v\nwant = %+v\n", got, want)
	}
}

func TestPathParams(t *testing.T) {
	var cases = []struct {
		url, tmpl, field, want string
	}{
		{"/1234", "/{a}", "a", "1234"},
		{"/v1/1234", "/v1/{a}", "a", "1234"},
		{"/v1/user/5/home", "/v1/user/{userid}/home", "userid", "5"},
	}

	for _, test := range cases {
		ret, err := PathParams(test.url, test.tmpl)
		if err != nil {
			t.Errorf("PathParams returned error '%v' on case '%+v'\n", err, test)
		}
		if got, ok := ret[test.field]; ok {
			if got != test.want {
				t.Errorf("PathParams got '%v', want '%v'\n", got, test.want)
			}
		} else {
			t.Errorf("PathParams didn't return map containing field '%v'\n", test.field)
		}
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
