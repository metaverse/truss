package httptransport

import (
	"testing"
)

func TestPathExtract(t *testing.T) {
	var cases = []struct {
		url, tmpl, field, want string
	}{
		{"/1234", "/{a}", "a", "1234"},
		{"/v1/1234", "/v1/{a}", "a", "1234"},
		{"/v1/user/5/home", "/v1/user/{userid}/home", "userid", "5"},
	}

	for _, test := range cases {
		got, err := PathExtract(test.url, test.tmpl, test.field)
		if err != nil {
			t.Errorf("PathExtract returned error '%v' on case '%+v'\n", err, test)
		}
		if got != test.want {
			t.Errorf("PathExtract got '%v', want '%v'\n", got, test.want)
		}
	}
}

func TestGetSourceCode(t *testing.T) {
	file, err := GetSourceCode(PathExtract)
	if err != nil {
		t.Fatalf("Failed to get source code: %s\n", err)
	}
	t.Logf("%v\n", file)
}
