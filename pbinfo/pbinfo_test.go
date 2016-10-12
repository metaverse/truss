package pbinfo

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewCatalog(t *testing.T) {
	gf, err := os.Open("./test-go.txt")
	if err != nil {
		t.Fatal(err)
	}
	pf, err := os.Open("./test-proto.txt")
	if err != nil {
		t.Fatal(err)
	}

	cat, err := New([]io.Reader{gf}, []io.Reader{pf})
	if err != nil {
		t.Fatal(err)
	}
	if cat == nil {
		t.Fatal("returned catalog is nil!")
	}

	return
}

func TestTypeResolution(t *testing.T) {
	caseCode := `
package TEST
type EnumType int32

type NestedMessageA struct {
	A *NestedMessageC
}
type NestedMessageB struct {
	A []*NestedMessageC
}
type NestedMessageC struct {
	A int64
}

type NestedTypeRequest struct {
	A *NestedMessageA
	B []*NestedMessageB
	C EnumType
}`
	cat, err := New([]io.Reader{strings.NewReader(caseCode)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	//sp := spew.ConfigState{
	//Indent: "   ",
	//}
	tmap := newTypeMap(cat)
	//sp.Dump(cat)

	var cases = []struct {
		name, fieldname, typename string
	}{
		{"NestedMessageA", "A", "NestedMessageC"},
		{"NestedMessageB", "A", "NestedMessageC"},
		{"NestedTypeRequest", "A", "NestedMessageA"},
		{"NestedTypeRequest", "B", "NestedMessageB"},
		{"NestedTypeRequest", "C", "EnumType"},
	}
	for _, c := range cases {
		box, ok := tmap[c.name]
		if !ok {
			t.Errorf("Could not find %q in map of types", c.name)
		}
		msg := box.Message
		if msg.Name != c.name {
			t.Errorf("Message in typemap is named %q, wanted %q", msg.Name, c.name)
		}
		var selectedf *Field
		foundfield := false
		for _, f := range msg.Fields {
			if f.Name == c.fieldname {
				foundfield = true
				selectedf = f
			}
		}
		if !foundfield {
			t.Fatalf("Could't find field %q in message %q", c.fieldname, msg.Name)
		}

		ftypebox, ok := tmap[selectedf.Type.Name]
		if !ok {
			t.Errorf("Field %q has type %q which is not found in the typemap", selectedf.Name, selectedf.Type.Name)
		}
		if selectedf.Type.Enum != nil {
			ftype := ftypebox.Enum
			if selectedf.Type.Enum != ftype {
				t.Errorf("Field %q on message %q has type which differs from the typemap type of the same name, got %p, want %p", selectedf.Name, msg.Name, selectedf.Type, ftype)
			}
		}
	}
}

// Test that type resolution of map values functions correctly. So if a message
// has a map field, and that map field has values that are some other message
// type, then the type of the key will be correct.
func TestMapTypeResolution(t *testing.T) {
	caseCode := `
package TEST

type NestedMessageC struct {
	A int64
}
type MapNestedMsg struct {
	Beta map[int64]*NestedMessageC
}
`
	cat, err := New([]io.Reader{strings.NewReader(caseCode)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tmap := newTypeMap(cat)

	expected, ok := tmap["MapNestedMsg"]
	if !ok {
		t.Fatal("Couldn't find message 'MapNestedMsg'")
	}

	beta := expected.Message.Fields[0].Type.Map

	if beta.ValueType.Message != tmap["NestedMessageC"].Message {
		t.Fatalf("Expected beta ValueType to be 'NestedMessageC', is %q", beta.ValueType.Message.Name)
	}

}
