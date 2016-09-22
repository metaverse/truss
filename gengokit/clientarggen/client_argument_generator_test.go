package clientarggen

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gengokit/gentesthelper"
)

var (
	spw = spew.ConfigState{
		Indent: "   ",
	}
)

func TestNewClientServiceArgs(t *testing.T) {
	svc := deftree.ProtoService{
		Name: "AddSvc",
		Methods: []*deftree.ServiceMethod{
			&deftree.ServiceMethod{
				Name: "Sum",
				RequestType: &deftree.ProtoMessage{
					Name: "SumRequest",
					Fields: []*deftree.MessageField{
						&deftree.MessageField{
							Name:   "a",
							Number: 1,
							Label:  "LABEL_REPEATED",
							Type: deftree.FieldType{
								Name: "TYPE_INT64",
							},
						},
						&deftree.MessageField{
							Name:   "b",
							Number: 2,
							Label:  "LABEL_OPTIONAL",
							Type: deftree.FieldType{
								Name: "TYPE_INT64",
							},
						},
					},
				},
				ResponseType: &deftree.ProtoMessage{
					Name: "SumReply",
					Fields: []*deftree.MessageField{
						&deftree.MessageField{
							Name:   "v",
							Number: 1,
							Label:  "LABEL_OPTIONAL",
							Type: deftree.FieldType{
								Name: "TYPE_INT64",
							},
						},
						&deftree.MessageField{
							Name:   "err",
							Number: 2,
							Label:  "LABEL_OPTIONAL",
							Type: deftree.FieldType{
								Name: "TYPE_STRING",
							},
						},
					},
				},
			},
		},
	}
	csa := New(&svc)

	expected := &ClientServiceArgs{
		MethArgs: map[string]*MethodArgs{
			"Sum": &MethodArgs{
				Args: []*ClientArg{
					&ClientArg{
						Name:            "a",
						FlagName:        "sum.a",
						FlagArg:         "flagASum",
						FlagType:        "string",
						FlagConvertFunc: "flagASum = flag.String(\"sum.a\", \"\", \"\")",
						GoArg:           "ASum",
						GoType:          "int64",
						GoConvertInvoc:  "ASum := CarveASum(flagASum)",
						GoConvertFunc:   "\nfunc CarveASum(inpt string) []int64 {\n\tinpt = strings.Trim(inpt, \"[] \")\n\tslc := strings.Split(inpt, \",\")\n\tvar rv []int64\n\n\tfor _, item := range slc {\n\t\titem = strings.Trim(item, \" \")\n\t\titem = strings.Replace(item, \"'\", \"\\\"\", -1)\n\t\tif len(item) == 0 {\n\t\t\tcontinue\n\t\t}\n\t\ttmp, err := strconv.ParseInt(item, 10, 64)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Sprintf(\"couldn't parse '%v' of '%v'\", item, inpt))\n\t\t}\n\t\trv = append(rv, tmp)\n\t}\n\treturn rv\n}\n\n",
						ProtbufType:     "TYPE_INT64",
						IsBaseType:      true,
						Repeated:        true,
					},
					&ClientArg{

						Name:            "b",
						FlagName:        "sum.b",
						FlagArg:         "flagBSum",
						FlagType:        "int64",
						FlagConvertFunc: "flagBSum = flag.Int64(\"sum.b\", 0, \"\")",
						GoArg:           "BSum",
						GoType:          "int64",
						GoConvertInvoc:  "BSum := CarveBSum(flagBSum)",
						GoConvertFunc:   "\nfunc CarveBSum(inpt int64) []int64 {\n\tinpt = strings.Trim(inpt, \"[] \")\n\tslc := strings.Split(inpt, \",\")\n\tvar rv []int64\n\n\tfor _, item := range slc {\n\t\titem = strings.Trim(item, \" \")\n\t\titem = strings.Replace(item, \"'\", \"\\\"\", -1)\n\t\tif len(item) == 0 {\n\t\t\tcontinue\n\t\t}\n\t\ttmp, err := strconv.ParseInt(item, 10, 64)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Sprintf(\"couldn't parse '%v' of '%v'\", item, inpt))\n\t\t}\n\t\trv = append(rv, tmp)\n\t}\n\treturn rv\n}\n\n",
						ProtbufType:     "TYPE_INT64",
						IsBaseType:      true,
						Repeated:        false,
					},
				},
			},
		},
	}
	if got, want := csa, expected; !reflect.DeepEqual(got, want) {
		t.Errorf(gentesthelper.DiffStrings(spw.Sdump(got), spw.Sdump(want)))
	}
}
