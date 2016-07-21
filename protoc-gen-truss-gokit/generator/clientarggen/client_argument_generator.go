package clientarggen

import (
	"fmt"
	"strings"

	//log "github.com/Sirupsen/logrus"
	"github.com/TuneLab/gob/gendoc/doctree"
)

// A collection of the necessary information for generating basic business
// logic in the client. This business logic will allow the generated client to:
//     1. Have command line flags of the correct go types
//     2. Place correctly named and typed arguments in each request method
//     3. Create a request struct with the the function arguments as fields
//
// Since we can only automatically generate handlers for some basic types, if a
// ClientArg is for a field that's not a base type (such as if it's an embedded
// message) then the developer is going to have to write a handler for that
// themselves.
type ClientArg struct {
	Name            string
	FlagName        string
	FlagArg         string
	ProtbufType     string
	GoType          string
	FlagConvertFunc string
	IsBaseType      bool
}

type MethodArgs struct {
	Args []*ClientArg
}

// FunctionArgs returns a string for the arguments of the function signature
// for a given method. E.g. If there where a method "Sum" with arguments "A"
// and "B", this would generate the arguments portion of a function declaration
// like this:
//
//     func Sum(ASum int64, BSum int64) (pb.SumRequest, error) {
//              └────────────────────┘
func (self *MethodArgs) FunctionArgs() string {
	tmp := []string{}
	for _, a := range self.Args {
		tmp = append(tmp, fmt.Sprintf("%s %s", a.FlagArg, a.GoType))
	}
	return strings.Join(tmp, ", ")
}

// CallArgs returns a string for the variables to pass to a function
// implementing this method. Example:
//
//     request, _ := clientHandler.Sum(*ASum, *BSum)
//                                     └──────────┘
func (self *MethodArgs) CallArgs() string {
	tmp := []string{}
	for _, a := range self.Args {
		tmp = append(tmp, fmt.Sprintf("*%s", a.FlagArg))
	}
	return strings.Join(tmp, ", ")
}

type ClientServiceArgs struct {
	MethArgs map[string]*MethodArgs
}

func (self *ClientServiceArgs) AllFlags() string {
	tmp := []string{}
	for _, m := range self.MethArgs {
		for _, a := range m.Args {
			tmp = append(tmp, a.FlagConvertFunc)
		}
	}
	return strings.Join(tmp, "\n")
}

var ProtoToGoTypeMap = map[string]string{
	"TYPE_DOUBLE":   "float64",
	"TYPE_FLOAT":    "float32",
	"TYPE_INT64":    "int64",
	"TYPE_UINT64":   "uint64",
	"TYPE_INT32":    "int32",
	"TYPE_UINT32":   "uint32",
	"TYPE_FIXED64":  "uint64",
	"TYPE_FIXED32":  "uint32",
	"TYPE_BOOL":     "bool",
	"TYPE_STRING":   "string",
	"TYPE_SFIXED32": "int32",
	"TYPE_SFIXED64": "int64",
	"TYPE_SINT32":   "int32",
	"TYPE_SINT64":   "int64",
}

func New(svc *doctree.ProtoService) *ClientServiceArgs {
	svcArgs := ClientServiceArgs{
		MethArgs: make(map[string]*MethodArgs),
	}
	for _, meth := range svc.Methods {
		m := MethodArgs{}
		for _, field := range meth.RequestType.Fields {
			newArg := ClientArg{}
			newArg.Name = field.GetName()
			newArg.FlagName = fmt.Sprintf("%s.%s", strings.ToLower(meth.GetName()), strings.ToLower(field.GetName()))
			newArg.FlagArg = fmt.Sprintf("%s%s", strings.Title(newArg.Name), strings.Title(meth.GetName()))
			newArg.ProtbufType = field.Type.GetName()

			var gt string
			var ok bool
			if gt, ok = ProtoToGoTypeMap[field.Type.GetName()]; !ok {
				gt = "string"
				newArg.IsBaseType = false
			} else {
				newArg.IsBaseType = true
			}
			newArg.GoType = gt

			newArg.FlagConvertFunc = createFlagConvertFunc(newArg)

			m.Args = append(m.Args, &newArg)
		}
		svcArgs.MethArgs[meth.GetName()] = &m
	}

	return &svcArgs
}

// createFlagConvertFunc creates the go string for the flag invocation to parse
// a command line argument into it's correct type
func createFlagConvertFunc(a ClientArg) string {
	fType := ""
	switch {
	case strings.Contains(a.GoType, "int32"):
		fType = "%s = flag.Int(\"%s\", 0, %s)"
	case strings.Contains(a.GoType, "int64"):
		fType = "%s = flag.Int64(\"%s\", 0, %s)"
	case strings.Contains(a.GoType, "bool"):
		fType = "%s = flag.Bool(\"%s\", false, %s)"
	case strings.Contains(a.GoType, "float"):
		fType = "%s = flag.Float64\"%s\", 0.0, %s)"
	case strings.Contains(a.GoType, "string"):
		fType = "%s = flag.String(\"%s\", \"\", %s)"
	}
	return fmt.Sprintf(fType, a.FlagArg, a.FlagName, `""`)
}
