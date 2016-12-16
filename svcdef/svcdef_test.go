package svcdef

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestSvcdef(t *testing.T) {
	gf, err := os.Open("./test-go.txt")
	if err != nil {
		t.Fatal(err)
	}
	pf, err := os.Open("./test-proto.txt")
	if err != nil {
		t.Fatal(err)
	}

	sd, err := New(map[string]io.Reader{"./test-go.txt": gf}, map[string]io.Reader{"./test-proto.txt": pf})
	if err != nil {
		t.Fatal(err)
	}
	if sd == nil {
		t.Fatal("returned SvcDef is nil!")
	}
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
	sd, err := New(map[string]io.Reader{"/tmp/notreal": strings.NewReader(caseCode)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tmap := newTypeMap(sd)

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

// Test that after calling New(), type resolution of map values functions
// correctly. So if a message has a map field, and that map field has values
// that are some other message type, then the type of the key will be correct.
func TestNewMapTypeResolution(t *testing.T) {
	caseCode := `
package TEST

type NestedMessageC struct {
	A int64
}
type MsgWithMap struct {
	Beta map[int64]*NestedMessageC
}
`
	sd, err := New(map[string]io.Reader{"/tmp/notreal": strings.NewReader(caseCode)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// findMsg defined here for brevity
	findMsg := func(name string) *Message {
		for _, m := range sd.Messages {
			if m.Name == name {
				return m
			}
		}
		return nil
	}

	msg := findMsg("MsgWithMap")
	if msg == nil {
		t.Fatal("Couldn't find message 'MsgWithMap'")
	}
	expected := findMsg("NestedMessageC")
	if expected == nil {
		t.Fatal("Couldn't find message 'NestedMessageC'")
	}

	beta := msg.Fields[0].Type.Map

	if beta.ValueType.Message != expected {
		t.Fatalf("Expected beta ValueType to be 'NestedMessageC', is %q", beta.ValueType.Message.Name)
	}

}
