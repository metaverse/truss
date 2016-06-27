package svcparse

import (
	//"fmt"
	"strings"
	"testing"
)

func TestScanReadUnit(t *testing.T) {
	r := strings.NewReader("what\nservice service Test{}")
	scn := NewSvcScanner(r)
	for i := 0; i < 10; i++ {
		out, err := scn.ReadUnit()
		if err != nil {
			break
		}
		t.Logf("%v unit: '%v'\n", i, string(out))
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
}
