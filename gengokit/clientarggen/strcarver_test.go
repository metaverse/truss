package clientarggen

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TuneLab/go-truss/gengokit/gentesthelper"
)

var _ = gentesthelper.FuncSourceCode

type Carver func(string) interface{}

// runCarveTest is a generic test runner for testing the functionality of
// the generated slice parsing functions.
func runCarveTest(arg ClientArg, artisan interface{}, cases []string, expected interface{}, t *testing.T) {
	generated := GenerateCarveFunc(&arg)
	raw, err := gentesthelper.FuncSourceCode(artisan)
	if err != nil {
		t.Errorf("couldn't retrieve source for '%v': '%v'", artisan, err)
	}
	// Use TrimSpace to normalize raw and generated output
	if got, want := strings.TrimSpace(generated), strings.TrimSpace(raw); got != want {
		t.Error(gentesthelper.DiffStrings(got, want))
	}

	cv := reflect.ValueOf(artisan)
	for _, item := range cases {
		cinpt := reflect.ValueOf(item)
		if got, want := cv.Call([]reflect.Value{cinpt})[0].Interface(), expected; !reflect.DeepEqual(got, want) {
			t.Errorf("Got '%v', expected '%v'", got, want)
		}
	}
}

func TestCarveInteger32(t *testing.T) {
	// Test that the generated source code matches the source code in this
	// module that we test against
	c := ClientArg{
		Name:     "integerthirtytwo",
		FlagType: "string",
		GoArg:    "Integerthirtytwo",
		GoType:   "[]int32",
	}
	expected := []int32{12, 13, 2, 1, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5]",
		"[12,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveIntegerthirtytwo, cases, expected, t)
}

func TestCarveInteger64(t *testing.T) {
	c := ClientArg{
		Name:     "integersixtyfour",
		FlagType: "string",
		GoArg:    "Integersixtyfour",
		GoType:   "[]int64",
	}
	expected := []int64{12, 13, 2, 1, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5]",
		"[12,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveIntegersixtyfour, cases, expected, t)
}

func TestCarveUint32(t *testing.T) {
	c := ClientArg{
		Name:     "uintthirtytwo",
		FlagType: "string",
		GoArg:    "Uintthirtytwo",
		GoType:   "[]uint32",
	}
	expected := []uint32{12, 13, 2, 1, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5]",
		"[12,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveUintthirtytwo, cases, expected, t)
}

func TestCarveUint64(t *testing.T) {
	c := ClientArg{
		Name:     "uintsixtyfour",
		FlagType: "string",
		GoArg:    "Uintsixtyfour",
		GoType:   "[]uint64",
	}
	expected := []uint64{12, 13, 2, 1, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5]",
		"[12,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveUintsixtyfour, cases, expected, t)
}

func TestCarveFloat32(t *testing.T) {
	c := ClientArg{
		Name:     "floatthirtytwo",
		FlagType: "string",
		GoArg:    "Floatthirtytwo",
		GoType:   "[]float32",
	}
	expected := []float32{12.0, 13.0, 2.0, 1.0, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5.0]",
		"[12.0,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveFloatthirtytwo, cases, expected, t)
}

func TestCarveFloat64(t *testing.T) {
	c := ClientArg{
		Name:     "floatsixtyfour",
		FlagType: "string",
		GoArg:    "Floatsixtyfour",
		GoType:   "[]float64",
	}
	expected := []float64{12.0, 13.0, 2.0, 1.0, 4, 5}
	cases := []string{
		"12,13,2,1, 4, 5",
		"[12, 13, 2, 1, 4, 5.0]",
		"[12.0,13,2,1,4,5]",
		// Unbalanced
		"[12,13,2,1, 4, 5",
	}
	runCarveTest(c, CarveFloatsixtyfour, cases, expected, t)
}

func TestCarveBool(t *testing.T) {
	c := ClientArg{
		Name:     "bool",
		FlagType: "string",
		GoArg:    "Bool",
		GoType:   "[]bool",
	}
	expected := []bool{true, false, false, true, true}
	cases := []string{
		"true,false,false,true, true,",
		"[true, false, false, true, true,]",
		"[true,false,false,true,true,]",
		// Unbalanced
		"[true,false,false,true, true,",
	}
	runCarveTest(c, CarveBool, cases, expected, t)
}

func TestCarveString(t *testing.T) {
	c := ClientArg{
		Name:     "string",
		FlagType: "string",
		GoArg:    "String",
		GoType:   "[]string",
	}
	expected := []string{"foo", "Bar", "kaboom!", "BANG", "example"}
	cases := []string{
		// Single quotes around a each sub string
		`'foo', 'Bar', 'kaboom!', 'BANG', 'example'`,
		// Double quotes around each sub string
		`"foo", "Bar", "kaboom!", "BANG", "example"`,
		`"foo","Bar","kaboom!","BANG","example"`,
		`["foo", "Bar", "kaboom!", "BANG", "example"]`,
		`["foo","Bar","kaboom!","BANG","example"]`,
		// Unbalanced
		`["foo", "Bar", "kaboom!", "BANG", "example"`,
	}
	runCarveTest(c, CarveString, cases, expected, t)
}
