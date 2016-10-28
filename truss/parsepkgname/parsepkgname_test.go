package parsepkgname

import (
	"io"
	"strings"
	"testing"
)

type testScanner struct {
	contents [][]rune
	position int
}

func (t *testScanner) ReadUnit() ([]rune, error) {
	if t.position < len(t.contents) {
		rv := t.contents[t.position]
		t.position += 1
		return rv, nil
	}
	return nil, io.EOF
}

func NewTestScanner(units []string) *testScanner {
	rv := testScanner{position: 0}
	for _, u := range units {
		rv.contents = append(rv.contents, []rune(u))
	}
	return &rv
}

func TestFromScanner_simple(t *testing.T) {
	basicContents := []string{
		"\n",
		"package",
		" ",
		"examplename",
		";",
		"\n",
	}
	scn := NewTestScanner(basicContents)
	want := "examplename"
	got, err := FromScanner(scn)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Got %q for package name, want %q", got, want)
	}
}

func TestFromScanner_mid_comment(t *testing.T) {
	contents := []string{
		"\n",
		"package",
		" ",
		"/* comment in the middle of the declaration */",
		"examplename",
		";",
		"\n",
	}
	scn := NewTestScanner(contents)
	want := "examplename"
	got, err := FromScanner(scn)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Got %q for package name, want %q", got, want)
	}
}

func TestFromReader(t *testing.T) {
	code := `
// A comment about this proto file
package /* some mid-definition comment */ examplepackage;

// and the rest of the file goes here
`
	name, err := FromReader(strings.NewReader(code))
	if err != nil {
		t.Fatal(err)
	}
	got := name
	want := "examplepackage"
	if got != want {
		t.Fatalf("Got %q for package name, want %q", got, want)
	}
}
