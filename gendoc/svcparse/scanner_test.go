package svcparse

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func cleanStr(s string) string {
	cleanval := strings.Replace(s, "\n", "\\n", -1)
	cleanval = strings.Replace(cleanval, "\t", "\\t", -1)
	cleanval = strings.Replace(cleanval, "\"", "\\\"", -1)
	return cleanval
}

func TestBodyDetection(t *testing.T) {
	r := strings.NewReader(`
service Example_Service {
	rpc Example(Empty) returns (Empty) {
		option (google.api.http) = {
			// Some example comment
			get: "/ExampleGet"
			body: "*"

			additional_bindings {
				post: "/ExamplePost"
			}
			// Testing comments
		}
	}
	// More comment handling
}
// Handle these comments!
`)
	scn := NewSvcScanner(r)

	for {
		err := scn.FastForward()
		if err != nil {
			break
		}
		unit, err := scn.ReadUnit()
		if err != nil {
			break
		}
		t.Logf("%16.16v %6v %6v %4v\n", cleanStr(string(unit)), scn.InBody, scn.InDefinition, scn.BraceLevel)
	}
	for i := 0; i < 10; i++ {
		scn.UnreadUnit()
	}
	for {
		err := scn.FastForward()
		if err != nil {
			break
		}
		unit, err := scn.ReadUnit()
		if err != nil {
			break
		}
		t.Logf("%16.16v %6v %6v %4v\n", cleanStr(string(unit)), scn.InBody, scn.InDefinition, scn.BraceLevel)
	}
}

func TestScanSingleLineComments(t *testing.T) {
	r := strings.NewReader("service testing\n // comment1\n //comment2\n\n//comment 3 \n what")
	scn := NewSvcScanner(r)
	for i, good_str := range []string{
		"service",
		" ",
		"testing",
		"\n ",
		"// comment1\n",
		" ",
		"//comment2\n",
		"\n",
		"//comment 3 \n",
		" ",
		"what",
		"",
	} {
		unit, err := scn.ReadUnit()
		str := string(unit)

		if str != good_str {
			for _, grp := range scn.Buf {
				t.Logf("  '%v'\n", cleanStr(string(grp.Value)))
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, cleanStr(str), cleanStr(good_str))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf(fmt.Sprintf("%v", err))
		}

	}
}
