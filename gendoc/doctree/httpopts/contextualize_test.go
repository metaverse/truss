package httpopts

import (
	"testing"

	dt "github.com/TuneLab/go-truss/gendoc/doctree"
)

func TestGetPathParams(t *testing.T) {
	binding := &dt.MethodHttpBinding{
		Fields: []*dt.BindingField{
			&dt.BindingField{
				Kind:  "get",
				Value: `"/{a}/{b}"`,
			},
		},
	}
	params := getPathParams(binding)
	t.Log(params)
	if len(params) != 2 {
		t.Fatalf("Params (%v) is length '%v', expected length 2", params, len(params))
	}
}

// Make sure that the location of fields in HTTP parameters matches up with the
// locations specified within MethodHttpBinding
func TestPostBodyParams(t *testing.T) {
	typ := dt.FieldType{}
	typ.SetName("TYPE_STRING")

	msg := &dt.ProtoMessage{
		Fields: []*dt.MessageField{
			&dt.MessageField{
				Number: 1,
				Label:  "LABEL_OPTIONAL",
				Type:   typ,
			},
			&dt.MessageField{
				Number: 2,
				Label:  "LABEL_OPTIONAL",
				Type:   typ,
			},
			&dt.MessageField{
				Number: 3,
				Label:  "LABEL_OPTIONAL",
				Type:   typ,
			},
		},
	}
	// In order to contextualize http bindings, there must be ProtoMessages
	// with `Name` fields which match the ones specified within the
	// BindingFields of the HttpBindings. However, since the `Name` field of
	// pretty much all the types in the Doctree module are actually fields on
	// embeded structs which aren't exported, we can't define the `Name` fields
	// of MessageField types inline. For this reason, the MessageFields are
	// defined above, but their names are set in the for loop below.
	for count, field := range msg.Fields {
		names := []string{"A", "B", "C"}
		field.SetName(names[count])
	}
	md := &dt.MicroserviceDefinition{
		Files: []*dt.ProtoFile{
			&dt.ProtoFile{
				Messages: []*dt.ProtoMessage{
					msg,
				},
				Services: []*dt.ProtoService{
					&dt.ProtoService{
						Methods: []*dt.ServiceMethod{
							&dt.ServiceMethod{
								RequestType:  msg,
								ResponseType: msg,
								HttpBindings: []*dt.MethodHttpBinding{
									&dt.MethodHttpBinding{
										Fields: []*dt.BindingField{
											&dt.BindingField{
												Kind:  "post",
												Value: `/{A}`,
											},
											&dt.BindingField{
												Kind:  "body",
												Value: `B`,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	Assemble(md)

	params := md.Files[0].Services[0].Methods[0].HttpBindings[0].Params
	if len(params) != 3 {
		t.Fatalf("Params (%s) has length %v, expected length of 3.\n", params, len(params))
	}

	for _, param := range params {
		switch param.Name {
		case "A":
			if param.Location != "path" {
				t.Fatalf("Expected param '%s' to have location 'path', instead has location '%s'\n", param, param.Location)
			}
		case "B":
			if param.Location != "body" {
				t.Fatalf("Expected param '%s' to have location 'body', instead has location '%s'\n", param, param.Location)
			}
		case "C":
			if param.Location != "query" {
				t.Fatalf("Expected param '%s' to have location 'query', instead has location '%s'\n", param, param.Location)
			}
		}
		t.Log(param)
	}
}
