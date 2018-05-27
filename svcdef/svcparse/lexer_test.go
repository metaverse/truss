package svcparse

import (
	//"fmt"
	"strings"
	"testing"
)

func TestScanReadUnit(t *testing.T) {

	r := strings.NewReader("what\nservice service Test{}")
	scn := NewSvcScanner(r)
	for _, good := range []string{"what", "\n", "service",
		" ", "service", " ", "Test", "{", "}"} {
		group, err := scn.ReadUnit()
		if err != nil {
			t.Fatalf("ReadUnit returned err: %v\n", err)
		}
		if string(group) != good {
			t.Fatalf("Returned unit '%v' differs from expected unit '%v'\n", string(group), good)
		}
	}
}

func TestScanFastForward(t *testing.T) {
	r := strings.NewReader("foo bar service baz")
	scn := NewSvcScanner(r)

	err := scn.FastForward()
	if err != nil {
		t.Fatalf("Error encountered on basic fast-forward: '%v'\n", err)
	}

	buf, err := scn.ReadUnit()
	if err != nil {
		t.Fatalf("Error on ReadUnit after basic FastForward: '%v'\n", err)
	}
	if got, want := string(buf), "service"; got != want {
		t.Fatalf("scn.ReadUnit() = '%v'; want '%v'\n", got, want)
	}

	buf, err = scn.ReadUnit()
	if err != nil {
		t.Fatalf("Error on ReadUnit: '%v'\n", err)
	}
	if got, want := string(buf), " "; got != want {
		t.Fatalf("scn.ReadUnit() = '%v'; want '%v'\n", got, want)
	}

	buf, err = scn.ReadUnit()
	if err != nil {
		t.Fatalf("Error on ReadUnit: '%v'\n", err)
	}
	if got, want := string(buf), "baz"; got != want {
		t.Fatalf("scn.ReadUnit() = '%v'; want '%v'\n", got, want)
	}
}

func TestScanStringLiterals(t *testing.T) {
	r := strings.NewReader(`ident"strlit"identStr "secstrlit" morestr`)
	scn := NewSvcScanner(r)

	for i, good_str := range []string{
		"ident",
		`"strlit"`,
		"identStr",
		" ",
		`"secstrlit"`,
		" ",
		"morestr",
	} {
		group, err := scn.ReadUnit()
		if err != nil {
			t.Fatalf("ReadUnit returned err: '%v'\n", err)
		}
		if string(group) != good_str {
			t.Fatalf("%v returned group '%v' differs from expected unit '%v'\n", i, string(group), good_str)
		}
	}
}

func TestBraceLevel(t *testing.T) {
	r := strings.NewReader("a{ c { }}")
	scn := NewSvcScanner(r)

	for i, good_lvl := range []int{0, 1, 1, 1, 1, 2, 2, 1, 0} {
		_, err := scn.ReadUnit()
		if err != nil {
			t.Logf("ReadUnit returned error: '%v'\n", err)
			t.Fail()
		}
		if good_lvl != scn.BraceLevel {
			t.Logf("Unexpected brace level on unit %v: Expected '%v', found '%v'\n", i, good_lvl, scn.BraceLevel)
			t.Fail()
		}
	}

	// Test bracelevel is correctly handled for quote escapes
	r = strings.NewReader("{ \"{\" }")
	scn = NewSvcScanner(r)

	for i, good_lvl := range []int{1, 1, 1, 1, 0} {
		_, err := scn.ReadUnit()
		if err != nil {
			t.Logf("ReadUnit returned error: '%v'\n", err)
			t.Fail()
		}
		if good_lvl != scn.BraceLevel {
			t.Logf("Unexpected brace level on unit %v: Expected '%v', found '%v'\n", i, good_lvl, scn.BraceLevel)
			t.Fail()
		}
	}
}

func TestLinNos(t *testing.T) {
	r := strings.NewReader("f\n\nj\nw\n\\n\nwhat")
	scn := NewSvcScanner(r)
	for i, good_linno := range []int{1, 3, 3, 4, 4, 5, 5, 5, 6, 6} {
		_, err := scn.ReadUnit()
		if err != nil {
			t.Logf("ReadUnit returned error: '%v'\n", err)
			t.Fail()
		}
		if good_linno != scn.GetLineNumber() {
			t.Logf("Unexpected line number on unit %v: Expected '%v', found '%v'\n", i, good_linno, scn.GetLineNumber())
			t.Fail()
		}
	}
}

func TestLexSingleLineComments(t *testing.T) {
	r := strings.NewReader("service testing\n // comment1\n//comment2\n\n//comment 3 \n what")
	lex := NewSvcLexer(r)
	for i, good_str := range []string{
		"service",
		" ",
		"testing",
		"\n ",
		"// comment1\n//comment2\n",
		"\n",
		"//comment 3 \n",
		" ",
		"what",
		"",
	} {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}

		if str != good_str {
			for _, grp := range lex.Buf {
				t.Logf("  '%v'\n", cleanStr(grp.value))
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, cleanStr(str), cleanStr(good_str))
		}

		if tk == EOF {
			break
		}
	}
}

func TestLexMultiLineComments(t *testing.T) {
	r := strings.NewReader("service testing\n /* comment1 */ /* thing */\n/*comment2 */\n\n/*comment 3 */\n what")
	lex := NewSvcLexer(r)
	for i, good_str := range []string{
		"service",
		" ",
		"testing",
		"\n ",
		"/* comment1 */ /* thing */",
		"\n",
		"/*comment2 */",
		"\n\n",
		"/*comment 3 */",
		"\n ",
		"what",
		"",
	} {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}

		if str != good_str {
			for _, grp := range lex.Buf {
				t.Logf("  '%v'\n", cleanStr(grp.value))
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, cleanStr(str), cleanStr(good_str))
		}

		if tk == EOF || tk == ILLEGAL {
			break
		}
	}
}

func TestLextUnGetToken(t *testing.T) {
	// Ensure that ungetting all the tokens doesn't cause them to change
	r := strings.NewReader("service testing\n // comment1\n//comment2\n\n//comment 3 \n what")
	lex := NewSvcLexer(r)

	good_vals := []string{
		"service",
		" ",
		"testing",
		"\n ",
		"// comment1\n//comment2\n",
		"\n",
		"//comment 3 \n",
		" ",
		"what",
		"",
	}

	for i, good_str := range good_vals {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}
		if str != good_str {
			for _, grp := range lex.Buf {
				t.Logf("  '%v'\n", grp.value)
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, str, good_str)
		}
	}
	for i := 0; i < len(good_vals); i++ {
		lex.UnGetToken()
	}
	for i, good_str := range good_vals {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}
		if str != good_str {
			for _, grp := range lex.Buf {
				t.Logf("  '%v'\n", grp.value)
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, str, good_str)
		}
	}

}

func TestLexNewlines(t *testing.T) {
	// Ensure that all newlines are properly accounted for
	r := strings.NewReader("service 1\n//2\n//3\n4\n5\n6")
	lex := NewSvcLexer(r)

	type val struct {
		v    string
		line int
	}

	good_vals := []val{
		{"service", 1},
		{" ", 1},
		{"1", 1},
		{"\n", 2},
		{"//2\n//3\n", 4},
		{"4", 4},
		{"\n", 5},
		{"5", 5},
		{"\n", 6},
		{"6", 6},
	}
	for i, g := range good_vals {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}
		if str != g.v {
			t.Log(lex.Buf)
			for _, grp := range lex.Buf {
				t.Logf("%v\n", grp)
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, str, g.v)
		}
		if lex.GetLineNumber() != g.line {
			t.Log(lex.Buf)
			t.Fatalf("Expected line '%v' go line '%v'\n", g.line, lex.GetLineNumber())
		}
	}
	for i := 0; i < 7; i++ {
		lex.UnGetToken()
	}
	for i, g := range good_vals[len(good_vals)-7:] {
		tk, str := lex.GetToken()
		if tk == ILLEGAL {
			t.Fatalf("Recieved ILLEGAL token on '%v' call to GetToken\n", i)
		}
		if str != g.v {
			t.Log(lex.Buf)
			for _, grp := range lex.Buf {
				t.Logf("%v\n", grp)
			}
			t.Fatalf("%v returned token '%v' differs from expected token '%v'\n", i, str, g.v)
		}
		if lex.GetLineNumber() != g.line {
			t.Log(lex.Buf)
			t.Fatalf("Expected line '%v' go line '%v'\n", g.line, lex.GetLineNumber())
		}
	}
}
