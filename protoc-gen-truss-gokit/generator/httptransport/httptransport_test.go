package httptransport

import (
	"testing"
)

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
	file, err := FuncSourceCode(PathParams)
	if err != nil {
		t.Fatalf("Failed to get source code: %s\n", err)
	}
	t.Logf("%v\n", file)
}

func TestAllFuncSourceCode(t *testing.T) {
	file, err := AllFuncSourceCode(PathParams)
	if err != nil {
		t.Fatalf("Failed to get source code: %s\n", err)
	}
	t.Logf("%v\n", file)
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
