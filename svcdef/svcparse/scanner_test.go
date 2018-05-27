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

func TestScanMultiLineComments(t *testing.T) {
	r := strings.NewReader("service testing\n /* comment1 */\n/*comment2 */\n\n/*comment 3 */\n what")
	scn := NewSvcScanner(r)
	for i, good_str := range []string{
		"service",
		" ",
		"testing",
		"\n ",
		"/* comment1 */",
		"\n",
		"/*comment2 */",
		"\n\n",
		"/*comment 3 */",
		"\n ",
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
			t.Fatalf("%v\n", err)
		}

	}
}
