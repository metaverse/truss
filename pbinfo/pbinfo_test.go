package pbinfo

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestNewCatalog(t *testing.T) {
	fmt.Println("testing!")
	gf, err := os.Open("./test-go.txt")
	if err != nil {
		t.Fatal(err)
	}
	pf, err := os.Open("./test-proto.txt")
	if err != nil {
		t.Fatal(err)
	}

	cat, err := New([]io.Reader{gf}, pf)
	if err != nil {
		t.Fatal(err)
	}
	if cat == nil {
		t.Fatal("returned catalog is nil!")
	}

	return
}
